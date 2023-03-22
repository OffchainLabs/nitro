package util

type Option[T any] struct {
	value *T
}

func None[T any]() Option[T] {
	return Option[T]{nil}
}

func Some[T any](x T) Option[T] {
	return Option[T]{&x}
}

func (x Option[T]) IsNone() bool {
	return x.value == nil
}

func (x Option[T]) Unwrap() T {
	return *x.value
}
