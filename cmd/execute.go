package cmd

import (
	"fmt"

	"github.com/alecthomas/kong"
	"go.uber.org/zap"
)

func Execute(
	args []string,
	logger *zap.Logger,
) error {
	cli := CLI{}

	parser, err := kong.New(&cli)
	if err != nil {
		return fmt.Errorf("could not create cli: %w", err)
	}

	context, err := parser.Parse(args)
	parser.FatalIfErrorf(err)

	err = context.Run(logger)
	parser.FatalIfErrorf(err)

	return nil
}
