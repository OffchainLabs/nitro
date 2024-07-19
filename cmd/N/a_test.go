package N

import (
	"math/rand"
	"testing"
)

func TestAliasing1(t *testing.T) {
	Fail(t, "fail")
}

func TestAliasing(t *testing.T) {
	if rand.Int()%2 == 0 {
		Fail(t, "fail")
	}
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
