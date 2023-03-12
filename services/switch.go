package services

import (
	"fmt"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/jtarchie/sqlite-tsdb/buffer"
	"github.com/jtarchie/sqlite-tsdb/sdk"
	"github.com/jtarchie/sqlite-tsdb/worker"
)

type Switcher struct {
	buffer    *buffer.Buffer[sdk.Event]
	count     uint64
	flushSize int
	path      string
	writer    *Writer
	worker    *worker.Worker[*Writer]
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
) (*Switcher, error) {
	writer, err := newNamedWriter(path)
	if err != nil {
		return nil, fmt.Errorf("could not create initial writer: %w", err)
	}

	switcher := &Switcher{
		buffer:    buffer.New[sdk.Event](bufferSize),
		count:     0,
		flushSize: flushSize,
		path:      path,
		writer:    writer,
		worker: worker.New(1, 1, func(i int, writer *Writer) {
			writer.Close()
			finalizer.Finalize(writer.Filename())
		}),
	}
	go switcher.process()

	return switcher, nil
}

func newNamedWriter(path string) (*Writer, error) {
	dbPath := filepath.Join(path, fmt.Sprintf("%d.db", time.Now().UnixNano()))

	return NewWriter(dbPath)
}

func (s *Switcher) process() {
	defer s.buffer.Close()

	for {
		event := s.buffer.Read()

		_ = s.writer.Insert(&event)

		current := atomic.AddUint64(&s.count, 1)
		if current%uint64(s.flushSize) == 0 {
			previousWriter := s.writer
			s.writer, _ = newNamedWriter(s.path)

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
