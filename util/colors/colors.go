// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package colors

import (
	"fmt"
	"regexp"
)

var Red = "\033[31;1m"
var Blue = "\033[34;1m"
var Yellow = "\033[33;1m"
var Pink = "\033[38;5;161;1m"
var Mint = "\033[38;5;48;1m"
var Grey = "\033[90m"

var Lime = "\033[38;5;119;1m"
var Lavender = "\033[38;5;183;1m"
var Maroon = "\033[38;5;124;1m"
var Orange = "\033[38;5;202;1m"

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

func PrintYellow(args ...interface{}) {
	print(Yellow)
	fmt.Print(args...)
	println(Clear)
}

func PrintPink(args ...interface{}) {
	print(Pink)
	fmt.Print(args...)
	println(Clear)
}

func Uncolor(text string) string {
	uncolor := regexp.MustCompile("\x1b\\[([0-9]+;)*[0-9]+m")
	unwhite := regexp.MustCompile(`\s+`)

	text = uncolor.ReplaceAllString(text, "")
	return unwhite.ReplaceAllString(text, " ")
}
