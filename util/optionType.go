package util

import "errors"

var ErrOptionIsEmpty = errors.New("option value is empty")

type Option[T any] struct {
	value *T
}

func EmptyOption[T any]() Option[T] {
	return Option[T]{nil}
}

func FullOption[T any](x T) Option[T] {
	return Option[T]{&x}
}

func (x Option[T]) IsEmpty() bool {
	return x.value == nil
}

func (x Option[T]) OpenKnownFull() T {
	return *x.value
}

func (x Option[T]) ToResult() Result[T] {
	if x.value == nil {
		return ErrorResult[T](ErrOptionIsEmpty)
	}
	return SuccessResult[T](*x.value)
}

func OptionMap[T, U any](o Option[T], f func(T) U) Option[U] {
	if o.value == nil {
		return Option[U]{nil}
	}
	u := f(*o.value)
	return Option[U]{&u}
}

type Result[T any] struct {
	err   error
	value T
}

func ErrorResult[T any](err error) Result[T] {
	return Result[T]{err: err}
}

func SuccessResult[T any](x T) Result[T] {
	return Result[T]{err: nil, value: x}
}

func (res Result[T]) Error() error {
	return res.err
}

func (res Result[T]) Open() (T, error) {
	return res.value, res.err
}

func (res Result[T]) OpenKnownSuccess() T {
	return res.value
}

func ResultMap[T, U any](res Result[T], f func(T) U) Result[U] {
	if res.err != nil {
		return ErrorResult[U](res.err)
	}
	return SuccessResult[U](f(res.value))
}
