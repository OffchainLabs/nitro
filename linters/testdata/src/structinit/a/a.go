// want package:"{package a .*}"
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
