package services

import (
	"fmt"
	"path/filepath"

	"github.com/c2fo/vfs/v6/vfssimple"
	"go.uber.org/zap"
)

type Persistence struct {
	remoteLocationPrefix string
	logger               *zap.Logger
}

func NewPersistence(
	remoteLocationPrefix string,
	logger *zap.Logger,
) *Persistence {
	return &Persistence{
		logger:               logger,
		remoteLocationPrefix: remoteLocationPrefix,
	}
}

func (p *Persistence) Finalize(filename string) {
	logger := p.logger

	localLocation := fmt.Sprintf("file://%s", filename)
	s3Location := fmt.Sprintf("%s/%s", p.remoteLocationPrefix, filepath.Base(filename))

	logger = logger.With(
		zap.String("remote", s3Location),
		zap.String("local", localLocation),
	)

	logger.Info("starting transfer")

	s3File, err := vfssimple.NewFile(s3Location)
	if err != nil {
		logger.Error("could not reference remote", zap.Error(err))

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
}
