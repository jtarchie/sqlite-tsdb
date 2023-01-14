package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/alecthomas/kong"
	"github.com/jtarchie/sqlite-tsdb/sdk"
	"github.com/jtarchie/sqlite-tsdb/server"
	"github.com/jtarchie/sqlite-tsdb/services"
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
		logger.Fatal("could not create logger", zap.Error(err))
		os.Exit(1)
	}

	err = execute(logger)
	if err != nil {
		logger.Fatal("could not execute", zap.Error(err))
		os.Exit(1)
	}
}

func execute(logger *zap.Logger) error {
	stats := sdk.StatsPayload{}

	_ = kong.Parse(&cli)

	dbPath := filepath.Join(cli.WorkPath, fmt.Sprintf("%d.db", time.Now().UnixNano()))

	dbService, err := services.NewDB(dbPath)
	if err != nil {
		return fmt.Errorf("could not start db service: %w", err)
	}

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
			logger.Error("could not read from body", zap.Error(err))

			return c.NoContent(http.StatusUnprocessableEntity)
		}
		defer body.Close()

		err = dbService.Insert(contents)
		if err != nil {
			logger.Error("could not capture event", zap.Error(err))

			return c.NoContent(http.StatusUnprocessableEntity)
		}

		return c.NoContent(http.StatusCreated)
	})

	e.GET("/api/stats", func(c echo.Context) error {
		return c.JSON(http.StatusOK, stats)
	})

	e.Logger.Fatal(e.Start(fmt.Sprintf(":%d", cli.Port)))

	return nil
}
