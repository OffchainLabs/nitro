package a

import "fmt"

// lint:require-exhaustive-initialization
type interestingStruct struct {
	x int
	b *boringStruct
}

type boringStruct struct {
	x, y int
}

func init() {
	a := &interestingStruct{ // Error: only single field is initialized.
		x: 1,
	}
	fmt.Println(a)
	b := interestingStruct{ // Error: only single field is initialized.
		b: nil,
	}
	fmt.Println(b)
	c := interestingStruct{ // Not an error, all fields are initialized.
		x: 1,
		b: nil,
	}
	fmt.Println(c)
	d := &boringStruct{ // Not an error since it's not annotated for the linter.
		x: 1,
	}
	fmt.Println(d)
}
