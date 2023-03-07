package buffer

type Buffer[T any] struct {
	input  chan T
	output chan T
}

func New[T any](size int) *Buffer[T] {
	b := &Buffer[T]{
		input:  make(chan T),
		output: make(chan T, size),
	}
	go b.run()

	return b
}

func (b *Buffer[T]) run() {
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

func (b *Buffer[T]) Write(value T) {
	b.input <- value
}

func (b *Buffer[T]) Read() T {
	return <-b.output
}

func (b *Buffer[T]) Close() {
	close(b.input)
}
