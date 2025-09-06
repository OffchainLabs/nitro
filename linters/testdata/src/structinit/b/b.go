// want package:"{package b .*}"
package b

import (
	"fmt"

	"github.com/offchainlabs/nitro/linters/testdata/src/structinit/a"
)

// lint:require-exhaustive-initialization
type AnotherStruct struct {
	X int
}

func init() {
	var partiallyInitialized = &a.InterestingStruct{ // Error: only single field is initialized.
		X: 1,
	}
	fmt.Println(partiallyInitialized)
}
