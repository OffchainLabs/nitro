package chk_test

import (
	"fmt"
	"math"

	"github.com/offchainlabs/nitro/util/chk"
)

func PublicWith(x uint64) (uint64, error) {
	px, err := chk.NewPos64(x)
	if err != nil {
		return 0, err
	}
	return someOtherCalculationWith(px, someCalculationWith(px)).Val(), nil
}

func someCalculationWith(x chk.Pos64) chk.Pos64 {
	// Other complicated logic here.
	return safelyDoubleWith(x)
}

func someOtherCalculationWith(x, y chk.Pos64) chk.Pos64 {
	// Other complicated logic here.
	return safelyAddWith(x, y)
}

func safelyDoubleWith(x chk.Pos64) chk.Pos64 {
	if x.Val() > math.MaxUint64/2 {
		return chk.MustPos64(math.MaxUint64)
	}
	return chk.MustPos64(x.Val() / 2)
}

func safelyAddWith(x, y chk.Pos64) chk.Pos64 {
	if x.Val() > math.MaxUint64-y.Val() {
		return chk.MustPos64(math.MaxUint64)
	}
	return chk.MustPos64(x.Val() + y.Val())
}

func Example() {
	r, _ := Public(10)
	fmt.Println(r)
	// Output: 30
}
