package rightshift

import "fmt"

func doThing(v int) int {
	return 1 >> v // Error: Ln: 6
}

func calc() {
	val := 10
	fmt.Printf("%v", 1>>val) // Error: Ln 11
	_ = doThing(1 >> val)    // Error: Ln 12
	fmt.Printf("%v", 1<<val) // valid
}
