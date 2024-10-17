// Package chk supplies a set of checked types.
//
// These types can be used to avoid repeatedly checking the same checks
// on function and method arguments at multiple layers in your code's call
// stack.
//
// For exmpample, if you have a package which provides a public function which
// accepts a uint64, but inside that package you have other functions which all
// need to be able to be able to operate on strictly positive integers:
//
// Without the chk package you might write code like:
//
//	func Public(x uint64) (uint64, error) {
//		if x == 0 {
//			return 0, errors.New("x must be positive")
//		}
//		y, err := someCalculation(x)
//		if err != nil {
//			return 0, err
//		}
//		z, err := someOtherCalculation(x, y)
//		if err != nil {
//			return 0, err
//		}
//		return z, nil
//	}
//
//	func someCalculation(x uint64) (uint64, error) {
//		if x == 0 {
//			return 0, errors.New("x must be positive")
//		}
//		// Other complicated logic here.
//		return safelyDouble(x), nil
//	}
//
//	func someOtherCalculation(x, y uint64) (uint64, error) {
//		if x == 0 {
//			return 0, errors.New("x must be positive")
//		}
//		if y == 0 {
//			return 0, errors.New("y must be positive")
//		}
//		// Other complicated logic here.
//		return safelyAdd(x, y), nil
//	}
//
//	func safelyDouble(x uint64) uint64 {
//		if x > math.MaxUint64/2 {
//			return math.MaxUint64
//		}
//		return x * 2
//	}
//
//	func safelyAdd(x, y uint64) uint64 {
//		if x > math.MaxUint64-y {
//			return math.MaxUint64
//		}
//		return x + y
//	}
//
// This sort of code is annoying to write and maintain, but it is necessary to
// enusure that a coding error in the future doesn't introduce some other caller
// of one of the internal functions which aren't guarded by a check for a
// positive value.
//
// With the chk package you can write code like this:
//
//	func PublicWith(x uint64) (uint64, error) {
//		px, err := chk.NewPos64(x)
//		if err != nil {
//			return 0, err
//		}
//		return someOtherCalculationWith(px, someCalculationWith(px)).Val(), nil
//	}
//
//	func someCalculationWith(x chk.Pos64) chk.Pos64 {
//		// Other complicated logic here.
//		return safelyDoubleWith(x)
//	}
//
//	func someOtherCalculationWith(x, y chk.Pos64) chk.Pos64 {
//		// Other complicated logic here.
//		return safelyAddWith(x, y)
//	}
//
//	func safelyDoubleWith(x chk.Pos64) chk.Pos64 {
//		if x.Val() > math.MaxUint64/2 {
//			return chk.MustPos64(math.MaxUint64)
//		}
//		return chk.MustPos64(x.Val() / 2)
//	}
//
//	func safelyAddWith(x, y chk.Pos64) chk.Pos64 {
//		if x.Val() > math.MaxUint64-y.Val() {
//			return chk.MustPos64(math.MaxUint64)
//		}
//		return chk.MustPos64(x.Val() + y.Val())
//	}
//
// Of course, if you don't mind forcing clients of your package to depend on
// the chk package as well, you can just have your public funciton take a
// chk.Pos64 argument directly.
package chk

import (
	"errors"
)

type pos64 uint64

// Pos64 is a type which represents a positive uint64.
//
// The "zero" value of Pos64 is 1.
type Pos64 struct {
	uint64
}

// NewPos64 returns a new Pos64 with the given value.
//
// errors if v is 0.
func NewPos64(v uint64) (Pos64, error) {
	if v == 0 {
		return Pos64{}, errors.New("v must be positive. got: 0")
	}
	return Pos64{v}, nil
}

// MustPos64 returns a new Pos64 with the given value.
//
// panics if v is 0.
func MustPos64(v uint64) Pos64 {
	if v == 0 {
		panic("v must be positive. got: 0")
	}
	return Pos64{v}
}

// Val returns the value of the Pos64.
func (p Pos64) Val() uint64 {
	// The zero value of Pos64 is 1.
	if p.uint64 == 0 {
		return 1
	}
	return p.uint64
}
