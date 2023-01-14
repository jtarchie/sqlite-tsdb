package main

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/alecthomas/kong"
	"github.com/jtarchie/sqlite-tsdb/sdk"
	"github.com/jtarchie/sqlite-tsdb/server"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

var cli struct {
	Port      int    `help:"port for http server" required:""`
	FlushSize int    `help:"size of queue when to flush to s3"`
	WorkPath  string `type:"existingdir" help:"store database in directory" required:""`
	S3        struct {
		AccessKeyID     string
		SecretAccessKey string

		Bucket         string
		Endpoint       *url.URL
		ForcePathStyle bool
		Path           string
		Region         string
		SkipVerify     bool
	} `embed:"" prefix:"s3-" group:"s3"`
}

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("could not create logger: %s", err)
	}

	err = execute(logger)
	if err != nil {
		logger.Fatal("could not execute", zap.Error(err))
	}
}

func execute(logger *zap.Logger) error {
	stats := sdk.StatsPayload{}

	_ = kong.Parse(&cli)

	dbPath := filepath.Join(cli.WorkPath, fmt.Sprintf("%d.db", time.Now().UnixNano()))

	db, err := sql.Open(driverName, dbPath)
	if err != nil {
		return fmt.Errorf("could not open sqlite db %q: %w", dbPath, err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS payloads (
			id         INTEGER PRIMARY KEY,
			payload    TEXT NOT NULL,
			timestamp  INT GENERATED ALWAYS AS (payload->'$.timestamp') VIRTUAL,
			value      TEXT GENERATED ALWAYS AS (payload->'$.value') VIRTUAL
		);
		CREATE INDEX IF NOT EXISTS payloads_timestamp ON payloads(timestamp);
		CREATE VIRTUAL TABLE events USING fts5(value, content=payloads, content_rowid=id);
	`)
	if err != nil {
		return fmt.Errorf("could not create schema in %q: %w", dbPath, err)
	}

	insertEvent, err := db.Prepare(`INSERT INTO payloads (payload) VALUES (?);`)
	if err != nil {
		return fmt.Errorf("could not create prepared insert statement: %w", err)
	}
	defer insertEvent.Close()

	e := echo.New()
	e.Use(server.ZapLogger(logger))

	e.GET("/ping", func(c echo.Context) error {
		return c.String(http.StatusOK, `{"status":"OK"}`)
	})

	e.PUT("/api/events", func(c echo.Context) error {
		atomic.AddUint64(&stats.Count.Insert, 1)

		body := c.Request().Body

		contents, err := io.ReadAll(body)
		if err != nil {
			return fmt.Errorf("could not read from body: %w", err)
		}
		defer body.Close()

		_, err = insertEvent.Exec(contents)
		if err != nil {
			return fmt.Errorf("could not insert event: %w", err)
		}

		return c.NoContent(http.StatusCreated)
	})

	e.GET("/api/stats", func(c echo.Context) error {
		return c.JSON(http.StatusOK, stats)
	})

	e.Logger.Fatal(e.Start(fmt.Sprintf(":%d", cli.Port)))

	return nil
}
