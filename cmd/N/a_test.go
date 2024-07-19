package N

import (
	"testing"
)

func TestAliasing(t *testing.T) {
	Fail(t, "fail")
}

func Fail(t *testing.T, printables ...interface{}) {
	t.Helper()
	FailImpl(t, printables...)
}

var Red = "\033[31;1m"
var Clear = "\033[0;0m"

func FailImpl(t *testing.T, printables ...interface{}) {
	t.Helper()
	t.Fatal(Red, printables, Clear)
}
