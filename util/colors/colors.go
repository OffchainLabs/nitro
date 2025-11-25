// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package colors

import (
	"fmt"
)

var Red = "\033[31;1m"
var Blue = "\033[34;1m"
var Mint = "\033[38;5;48;1m"
var Grey = "\033[90m"


var Clear = "\033[0;0m"

func PrintBlue(args ...interface{}) {
	print(Blue)
	fmt.Print(args...)
	println(Clear)
}

func PrintGrey(args ...interface{}) {
	print(Grey)
	fmt.Print(args...)
	println(Clear)
}

func PrintMint(args ...interface{}) {
	print(Mint)
	fmt.Print(args...)
	println(Clear)
}

func PrintRed(args ...interface{}) {
	print(Red)
	fmt.Print(args...)
	println(Clear)
}

