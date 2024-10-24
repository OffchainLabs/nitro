package chk_test

import (
	"errors"
	"fmt"
	"math"
)

func Public(x uint64) (uint64, error) {
	if x == 0 {
		return 0, errors.New("x must be positive")
	}
	y, err := someCalculation(x)
	if err != nil {
		return 0, err
	}
	z, err := someOtherCalculation(x, y)
	if err != nil {
		return 0, err
	}
	return z, nil
}

func someCalculation(x uint64) (uint64, error) {
	if x == 0 {
		return 0, errors.New("x must be positive")
	}
	// Other complicated logic here.
	return safelyDouble(x), nil
}

func someOtherCalculation(x, y uint64) (uint64, error) {
	if x == 0 {
		return 0, errors.New("x must be positive")
	}
	if y == 0 {
		return 0, errors.New("y must be positive")
	}
	// Other complicated logic here.
	return safelyAdd(x, y), nil
}

func safelyDouble(x uint64) uint64 {
	if x > math.MaxUint64/2 {
		return math.MaxUint64
	}
	return x * 2
}

func safelyAdd(x, y uint64) uint64 {
	if x > math.MaxUint64-y {
		return math.MaxUint64
	}
	return x + y
}

func Example_without() {
	r, _ := Public(10)
	fmt.Println(r)
	// Output: 30
}
