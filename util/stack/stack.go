// Copyright 2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

package stack

import (
	"fmt"

	"github.com/ethereum/go-ethereum/log"
)

type Stack[T any] []T

func NewStack[T any]() *Stack[T] {
	return &Stack[T]{}
}

func (s *Stack[T]) Push(v T) {
	if s == nil {
		log.Warn("trying to push nil stack")
		return
	}
	*s = append(*s, v)
}

func (s *Stack[T]) Pop() (T, error) {
	if s == nil {
		var zeroVal T
		return zeroVal, fmt.Errorf("trying to pop nil stack")
	}
	if len(*s) == 0 {
		var zeroVal T
		return zeroVal, fmt.Errorf("trying to pop empty stack")
	}
	i := len(*s) - 1
	val := (*s)[i]
	*s = (*s)[:i]
	return val, nil
}
