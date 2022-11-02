package protocol

import "errors"

var ErrOptionIsEmpty = errors.New("option value is empty")

type Option[T any] struct {
	empty bool
	value T
}

func EmptyOption[T any]() Option[T] {
	return Option[T]{empty: true}
}

func FullOption[T any](x T) Option[T] {
	return Option[T]{empty: false, value: x}
}

func (x Option[T]) IsEmpty() bool {
	return x.empty
}

func (x Option[T]) Open() (T, bool) {
	return x.value, x.empty
}

func (x Option[T]) OpenOrError() (T, error) {
	if x.empty {
		return x.value, ErrOptionIsEmpty
	}
	return x.value, nil
}

func (x Option[T]) OpenKnownFull() T {
	return x.value
}

func OptionMap[T, U any](o Option[T], f func(T) U) Option[U] {
	if o.IsEmpty() {
		return EmptyOption[U]()
	}
	return FullOption[U](f(o.value))
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
