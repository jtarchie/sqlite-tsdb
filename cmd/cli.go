package cmd

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/c2fo/vfs/v6/backend"
	"github.com/c2fo/vfs/v6/backend/s3"
	"github.com/c2fo/vfs/v6/vfssimple"
	"github.com/jtarchie/sqlite-tsdb/sdk"
	"github.com/jtarchie/sqlite-tsdb/server"
	"github.com/jtarchie/sqlite-tsdb/services"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

type CLI struct {
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

func (cli *CLI) Run(logger *zap.Logger) error {
	stats := sdk.StatsPayload{}

	backend.Register(
		fmt.Sprintf("s3://%s", cli.S3.Bucket),
		s3.NewFileSystem().WithOptions(
			s3.Options{
				AccessKeyID:     cli.S3.AccessKeyID,
				SecretAccessKey: cli.S3.SecretAccessKey,
				Region:          cli.S3.Region,
				Endpoint:        cli.S3.Endpoint.String(),
				ForcePathStyle:  cli.S3.ForcePathStyle,
			},
		))

	dbPath := filepath.Join(cli.WorkPath, fmt.Sprintf("%d.db", time.Now().UnixNano()))

	writer, err := services.NewWriter(dbPath)
	if err != nil {
		return fmt.Errorf("could not create writer: %w", err)
	}

	e := echo.New()
	e.Use(server.ZapLogger(logger))

	e.GET("/ping", func(c echo.Context) error {
		//nolint: wrapcheck
		return c.String(http.StatusOK, `{"status":"OK"}`)
	})

	e.PUT("/api/events", func(c echo.Context) error {
		body := c.Request().Body

		contents, err := io.ReadAll(body)
		if err != nil {
			logger.Error("could not read from body", zap.Error(err))

			//nolint: wrapcheck
			return c.NoContent(http.StatusUnprocessableEntity)
		}
		defer body.Close()

		err = writer.Insert(contents)
		if err != nil {
			logger.Error("could not capture event", zap.Error(err))

			//nolint: wrapcheck
			return c.NoContent(http.StatusUnprocessableEntity)
		}

		count := atomic.AddUint64(&stats.Count.Insert, 1)

		// when the writer count reaches flush range
		if count%uint64(cli.FlushSize) == 0 {
			previousDBPath := writer.Filename()
			dbPath := filepath.Join(cli.WorkPath, fmt.Sprintf("%d.db", time.Now().UnixNano()))

			localLocation := fmt.Sprintf("file://%s", previousDBPath)
			s3Location := fmt.Sprintf("s3://%s/%s", cli.S3.Bucket, filepath.Base(previousDBPath))

			logger.Info(
				"copying local to s3",
				zap.String("local", localLocation),
				zap.String("s3", s3Location),
			)

			err := writer.Close()
			if err != nil {
				logger.Error("could not close writer",
					zap.Error(err),
					zap.String("filename", writer.Filename()),
				)
			}

			writer, err = services.NewWriter(dbPath)
			if err != nil {
				logger.Error("could not create writer",
					zap.Error(err),
					zap.String("filename", dbPath),
				)

				//nolint: wrapcheck
				return c.NoContent(http.StatusInternalServerError)
			}

			s3File, err := vfssimple.NewFile(s3Location)
			if err != nil {
				logger.Error(
					"could not point to s3",
					zap.Error(err),
					zap.String("s3", s3Location),
				)

				//nolint: wrapcheck
				return c.NoContent(http.StatusInternalServerError)
			}

			localFile, err := vfssimple.NewFile(localLocation)
			if err != nil {
				logger.Error(
					"could not point to local",
					zap.Error(err),
					zap.String("local", localLocation),
				)

				//nolint: wrapcheck
				return c.NoContent(http.StatusInternalServerError)
			}

			err = localFile.CopyToFile(s3File)
			if err != nil {
				logger.Error(
					"could not copy local to s3",
					zap.Error(err),
					zap.String("s3", s3Location),
					zap.String("local", localLocation),
				)

				//nolint: wrapcheck
				return c.NoContent(http.StatusInternalServerError)
			}
		}

		//nolint: wrapcheck
		return c.NoContent(http.StatusCreated)
	})

	e.GET("/api/stats", func(c echo.Context) error {
		//nolint: wrapcheck
		return c.JSON(http.StatusOK, stats)
	})

	e.Logger.Fatal(e.Start(fmt.Sprintf(":%d", cli.Port)))

	return nil
}
