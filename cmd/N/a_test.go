package N

import (
	"errors"
	"runtime/debug"
	"testing"
)

func TestEmptyCliConfig(t *testing.T) {
	err := errors.New("please retry")
	Require(t, err)
}

func Require(t *testing.T, err error, text ...interface{}) {
	t.Helper()
	RequireImpl(t, err, text...)
}

var Red = "\033[31;1m"
var Clear = "\033[0;0m"

func RequireImpl(t *testing.T, err error, printables ...interface{}) {
	t.Helper()
	if err != nil {
		t.Log(string(debug.Stack()))
		t.Fatal(Red, printables, err, Clear)
	}
}
