// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbmath

import "fmt"

// A simple moving average of a generic number type.
type MovingAverage[T Number] struct {
	buffer         []T
	bufferPosition int
	sum            T
}

func NewMovingAverage[T Number](period int) (*MovingAverage[T], error) {
	if period <= 0 {
		return nil, fmt.Errorf("MovingAverage period specified as %v but it must be positive", period)
	}
	return &MovingAverage[T]{
		buffer: make([]T, 0, period),
	}, nil
}

func (a *MovingAverage[T]) Update(value T) {
	period := cap(a.buffer)
	if period == 0 {
		return
	}
	if len(a.buffer) < period {
		a.buffer = append(a.buffer, value)
		a.sum += value
	} else {
		a.sum += value
		a.sum -= a.buffer[a.bufferPosition]
		a.buffer[a.bufferPosition] = value
		a.bufferPosition = (a.bufferPosition + 1) % period
	}
}

// Average returns the current moving average, or zero if no values have been added.
func (a *MovingAverage[T]) Average() T {
	if len(a.buffer) == 0 {
		return 0
	}
	return a.sum / T(len(a.buffer))
}
