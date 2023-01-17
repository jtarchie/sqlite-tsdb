package cmd

import (
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"sync/atomic"

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

	writer, err := services.NewSwitcher(
		cli.WorkPath,
		cli.FlushSize,
		func(dbPath string) {
			localLocation := fmt.Sprintf("file://%s", dbPath)
			s3Location := fmt.Sprintf("s3://%s/%s", cli.S3.Bucket, filepath.Base(dbPath))

			logger = logger.With(
				zap.String("s3", s3Location),
				zap.String("local", localLocation),
			)

			s3File, err := vfssimple.NewFile(s3Location)
			if err != nil {
				logger.Error("could not reference s3", zap.Error(err))

				return
			}

			localFile, err := vfssimple.NewFile(localLocation)
			if err != nil {
				logger.Error("could not reference local", zap.Error(err))

				return
			}

			err = localFile.CopyToFile(s3File)
			if err != nil {
				logger.Error("could not copy", zap.Error(err))

				return
			}
		},
	)
	if err != nil {
		return fmt.Errorf("could not create switcher: %w", err)
	}

	e := echo.New()
	e.Use(server.ZapLogger(logger))

	e.GET("/ping", func(c echo.Context) error {
		//nolint: wrapcheck
		return c.String(http.StatusOK, `{"status":"OK"}`)
	})

	e.PUT("/api/events", func(c echo.Context) error {
		event := &sdk.Event{}

		err := c.Bind(event)
		if err != nil {
			logger.Error("could not parse event JSON", zap.Error(err))

			//nolint: wrapcheck
			return c.NoContent(http.StatusUnprocessableEntity)
		}

		writer.Insert(event)
		atomic.AddUint64(&stats.Count.Insert, 1)

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
