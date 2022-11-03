package protocol

import (
	"errors"
	"math"
	"time"
)

type SecondsDuration uint64

const MaxSecondsDuration = SecondsDuration(math.MaxUint64)

var ErrUnderflow = errors.New("arithmetic underflow")

func (sd SecondsDuration) SaturatingAdd(sd2 SecondsDuration) SecondsDuration {
	sum := sd + sd2
	if sum < sd {
		// overflowed, so return maxuint
		return SecondsDuration(math.MaxUint64)
	}
	return sum
}

func (sd SecondsDuration) Sub(sd2 SecondsDuration) (SecondsDuration, error) {
	if sd < sd2 {
		return 0, ErrUnderflow
	}
	return sd - sd2, nil
}

type TimeReference interface {
	Get() SecondsDuration
}

type realTimeReference struct{}

func NewRealTimeReference() TimeReference {
	return realTimeReference{}
}

func (realTimeReference) Get() SecondsDuration {
	return SecondsDuration(time.Now().Unix())
}

type artificialTimeReference struct {
	current SecondsDuration
}

func newArtificialTimeReference() *artificialTimeReference {
	return &artificialTimeReference{0}
}

func (atr *artificialTimeReference) Get() SecondsDuration {
	return atr.current
}

func (atr *artificialTimeReference) Set(newVal SecondsDuration) {
	atr.current = newVal
}

func (atr *artificialTimeReference) Add(delta SecondsDuration) {
	atr.current = atr.current.SaturatingAdd(delta)
}
