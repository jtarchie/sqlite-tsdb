package main

import (
	"fmt"
	"net/http"
	"sync/atomic"

	"github.com/alecthomas/kong"
	"github.com/jtarchie/sqlite-tsdb/sdk"
	"github.com/labstack/echo/v4"
)

var cli struct {
	Port int `help:"port for http server" required:""`
}

func main() {
	stats := sdk.StatsPayload{}

	e := echo.New()
	e.GET("/ping", func(c echo.Context) error {
		return c.String(http.StatusOK, `{"status":"OK"}`)
	})

	e.PUT("/api/events", func(c echo.Context) error {
		atomic.AddUint64(&stats.Count.Insert, 1)

		return c.NoContent(http.StatusCreated)
	})

	e.GET("/api/stats", func(c echo.Context) error {
		return c.JSON(http.StatusOK, stats)
	})

	_ = kong.Parse(&cli)

	e.Logger.Fatal(e.Start(fmt.Sprintf(":%d", cli.Port)))
}
