package cmd

import (
	"fmt"
	"net/http"
	"net/url"
	"sync/atomic"

	"github.com/c2fo/vfs/v6/backend"
	"github.com/c2fo/vfs/v6/backend/s3"
	"github.com/jtarchie/sqlite-tsdb/sdk"
	"github.com/jtarchie/sqlite-tsdb/server"
	"github.com/jtarchie/sqlite-tsdb/services"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

type CLI struct {
	Port       int    `help:"port for http server" required:""`
	FlushSize  int    `help:"numbers of items to flush to large file store"`
	BufferSize int    `help:"size of in-memory buffer" default:"100"`
	WorkPath   string `type:"existingdir" help:"store database in directory" required:""`
	S3         struct {
		AccessKeyID     string `help:"access key to the s3 bucket"`
		SecretAccessKey string `help:"secret access key to the s3 bucket"`

		Bucket         string   `help:"name of the s3 bucket"`
		Endpoint       *url.URL `help:"full URL to s3 endpoint, not including bucket"`
		ForcePathStyle bool     `help:"force the path style of URLs, rather than sub-domains"`
		Path           string   `help:"path to store files on bucket"`
		Region         string   `help:"region for the s3 bucket (usually only for AWS)"`
		SkipVerify     bool     `help:"do not verify the SSL certs"`
	} `embed:"" prefix:"s3-" group:"s3" help:"where to store the sqlite databases"`
}

func (cli *CLI) Run(logger *zap.Logger) error {
	stats := sdk.StatsPayload{}

	cli.registerBucketAuth()

	writer, err := services.NewSwitcher(
		cli.WorkPath,
		cli.FlushSize,
		cli.BufferSize,
		services.NewPersistence(
			fmt.Sprintf("s3://%s", cli.S3.Bucket),
			logger,
		),
		logger,
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

func (cli *CLI) registerBucketAuth() {
	backend.Register(
		fmt.Sprintf("s3://%s", cli.S3.Bucket),
		s3.NewFileSystem().WithOptions(
			s3.Options{
				AccessKeyID:                 cli.S3.AccessKeyID,
				SecretAccessKey:             cli.S3.SecretAccessKey,
				Region:                      cli.S3.Region,
				Endpoint:                    cli.S3.Endpoint.String(),
				ForcePathStyle:              cli.S3.ForcePathStyle,
				DisableServerSideEncryption: true,
			},
		),
	)
}
