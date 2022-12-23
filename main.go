package main

import (
	"fmt"
	"net/http"

	"github.com/alecthomas/kong"
	"github.com/labstack/echo/v4"
)

var cli struct {
	Port int `help:"port for http server" required:""`
}

func main() {
	e := echo.New()
	e.GET("/ping", func(c echo.Context) error {
		return c.String(http.StatusOK, `{"status":"OK"}`)
	})

	_ = kong.Parse(&cli)

	e.Logger.Fatal(e.Start(fmt.Sprintf(":%d", cli.Port)))
}
