package services

import (
	"fmt"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/jtarchie/sqlite-tsdb/sdk"
)

type Switcher struct {
	buffer       *buffer[sdk.Event]
	closedWriter ClosedWriter
	count        uint64
	flushSize    int
	path         string
	writer       *Writer
}

type ClosedWriter interface {
	Finalize(string)
}

type ClosedWriterWrap func(string)

func (c ClosedWriterWrap) Finalize(s string) {
	c(s)
}

func NewSwitcher(
	path string,
	flushSize int,
	bufferSize int,
	closedWriter ClosedWriter,
) (*Switcher, error) {
	writer, err := newNamedWriter(path)
	if err != nil {
		return nil, fmt.Errorf("could not create initial writer: %w", err)
	}

	switcher := &Switcher{
		buffer:       NewBuffer[sdk.Event](bufferSize),
		closedWriter: closedWriter,
		count:        0,
		flushSize:    flushSize,
		path:         path,
		writer:       writer,
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

			go func() {
				previousWriter.Close()
				s.closedWriter.Finalize(previousWriter.Filename())
			}()
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
