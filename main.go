package main

import (
	"os"

	"github.com/jtarchie/sqlite-tsdb/cmd"
	"go.uber.org/zap"
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		logger.Fatal("could not create logger", zap.Error(err))
		os.Exit(1)
	}

	err = cmd.Execute(os.Args[1:], logger)
	if err != nil {
		logger.Fatal("could not execute", zap.Error(err))
		os.Exit(1)
	}
}
