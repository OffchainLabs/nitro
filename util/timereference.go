package util

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

type ArtificialTimeReference struct {
	current SecondsDuration
}

func NewArtificialTimeReference() *ArtificialTimeReference {
	return &ArtificialTimeReference{0}
}

func (atr *ArtificialTimeReference) Get() SecondsDuration {
	return atr.current
}

func (atr *ArtificialTimeReference) Set(newVal SecondsDuration) {
	atr.current = newVal
}

func (atr *ArtificialTimeReference) Add(delta SecondsDuration) {
	atr.current = atr.current.SaturatingAdd(delta)
}
