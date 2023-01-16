package services

type buffer[T any] struct {
	input  chan T
	output chan T
}

func NewBuffer[T any](size int) *buffer[T] {
	b := &buffer[T]{
		input:  make(chan T),
		output: make(chan T, size),
	}
	go b.run()

	return b
}

func (b *buffer[T]) run() {
	for v := range b.input {
	retry:
		select {
		case b.output <- v:
		default:
			<-b.output

			goto retry
		}
	}

	close(b.output)
}

func (b *buffer[T]) Write(value T) {
	b.input <- value
}

func (b *buffer[T]) Read() T {
	return <-b.output
}

func (b *buffer[T]) Close() {
	close(b.input)
}
