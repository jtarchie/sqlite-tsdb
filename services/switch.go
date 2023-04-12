package services

import (
	"fmt"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/jtarchie/sqlite-tsdb/buffer"
	"github.com/jtarchie/sqlite-tsdb/sdk"
	"github.com/jtarchie/worker"
	"go.uber.org/zap"
)

type Switcher struct {
	buffer    *buffer.Buffer[sdk.Event]
	count     uint64
	flushSize int
	logger    *zap.Logger
	path      string
	worker    *worker.Worker[*Writer]
	writer    *Writer
}

type Finalizer interface {
	Finalize(string)
}

type FinalizerWrap func(string)

func (c FinalizerWrap) Finalize(s string) {
	c(s)
}

func NewSwitcher(
	path string,
	flushSize int,
	bufferSize int,
	finalizer Finalizer,
	logger *zap.Logger,
) (*Switcher, error) {
	writer, err := newNamedWriter(path, logger)
	if err != nil {
		return nil, fmt.Errorf("could not create initial writer: %w", err)
	}

	workerQueue := 100

	switcher := &Switcher{
		buffer:    buffer.New[sdk.Event](bufferSize),
		count:     0,
		flushSize: flushSize,
		logger:    logger,
		path:      path,
		writer:    writer,
		worker: worker.New(workerQueue, 1, func(i int, writer *Writer) {
			logger.Info("worker start",
				zap.Int("worker", i),
				zap.String("filename", writer.Filename()),
			)
			writer.Close()
			finalizer.Finalize(writer.Filename())
		}),
	}
	go switcher.process()

	return switcher, nil
}

func newNamedWriter(path string, logger *zap.Logger) (*Writer, error) {
	dbPath := filepath.Join(path, fmt.Sprintf("%d.db", time.Now().UnixNano()))

	return NewWriter(dbPath, logger)
}

func (s *Switcher) process() {
	var err error

	defer s.buffer.Close()

	for {
		event := s.buffer.Read()

		_ = s.writer.Insert(&event)

		current := atomic.AddUint64(&s.count, 1)
		if current%uint64(s.flushSize) == 0 {
			previousWriter := s.writer

			s.writer, err = newNamedWriter(s.path, s.logger)
			if err != nil {
				s.logger.Error("could not init new writer", zap.Error(err))
			}

			s.worker.Enqueue(previousWriter)
		}
	}
}

func (s *Switcher) Insert(event *sdk.Event) {
	s.buffer.Write(*event)
}

func (s *Switcher) Count() uint64 {
	return atomic.LoadUint64(&s.count)
}

func (s *Switcher) Close() error {
	err := s.writer.Close()
	if err != nil {
		return fmt.Errorf("could not close writer: %w", err)
	}

	s.buffer.Close()

	return nil
}
