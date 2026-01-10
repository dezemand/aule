package eventhandler

type Broker interface {
}

type Handler[T any] func(Event[T]) error

func Emit[T any](broker Broker, event Event[T]) error {

	return nil
}

func On[T any](broker Broker, topic Topic[T], handler Handler[T]) error {

	return nil
}
