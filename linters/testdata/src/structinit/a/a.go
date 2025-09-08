// want package:"{package a .*structinit/a.* .*}"
// The comment above ensures that during tests, the `structinit` analyzer
// will produce a `Fact` about the `structinit/a` package (with some prefix
// and suffix in its path). Since the fact will be of type `*accumulatedFieldCounts`,
// we just match arbitrary pattern (.*) - it will be just some address.

package a

import "fmt"

// lint:require-exhaustive-initialization
type InterestingStruct struct {
	X int
	B *BoringStruct
}

type BoringStruct struct {
	X, Y int
}

func init() {
	a := &InterestingStruct{ // Error: only single field is initialized.
		X: 1,
	}
	fmt.Println(a)
	b := InterestingStruct{ // Error: only single field is initialized.
		B: nil,
	}
	fmt.Println(b)
	c := InterestingStruct{ // Not an error, all fields are initialized.
		X: 1,
		B: nil,
	}
	fmt.Println(c)
	d := &BoringStruct{ // Not an error since it's not annotated for the linter.
		X: 1,
	}
	fmt.Println(d)
}
