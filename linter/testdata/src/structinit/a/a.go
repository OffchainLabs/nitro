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
	a := &interestingStruct{
		x: 1, // Error: only single field is initialized.
	}
	fmt.Println(a)
	b := interestingStruct{
		b: nil, // Error: only single field is initialized.
	}
	fmt.Println(b)
	c := &boringStruct{
		x: 1, // Not an error since it's not annotated for the linter.
	}
	fmt.Println(c)
}
