// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbmath

// A simple moving average of a generic number type.
type MovingAverage[T Number] struct {
	period         int
	buffer         []T
	bufferPosition int
	sum            T
}

func NewMovingAverage[T Number](period int) *MovingAverage[T] {
	if period <= 0 {
		panic("MovingAverage period must be positive")
	}
	return &MovingAverage[T]{
		period: period,
		buffer: make([]T, 0, period),
	}
}

func (a *MovingAverage[T]) Update(value T) {
	if a.period <= 0 {
		return
	}
	if len(a.buffer) < a.period {
		a.buffer = append(a.buffer, value)
		a.sum += value
	} else {
		a.sum += value
		a.sum -= a.buffer[a.bufferPosition]
		a.buffer[a.bufferPosition] = value
		a.bufferPosition = (a.bufferPosition + 1) % a.period
	}
}

// Average returns the current moving average, or zero if no values have been added.
func (a *MovingAverage[T]) Average() T {
	if len(a.buffer) == 0 {
		return 0
	}
	return a.sum / T(len(a.buffer))
}
