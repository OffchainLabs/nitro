package N

import (
	"context"
	"testing"
	"time"
)

func TestAliasing1(t *testing.T) {
	time.Sleep(5 * time.Second)
	Fail(t, "fail")
}

func TestAliasing(t *testing.T) {
	t.Parallel()

	go func() {
		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		<-ctx.Done()
		Fail(t, "fail")
	}()
	time.Sleep(5 * time.Second)
}

func Fail(t *testing.T, printables ...interface{}) {
	t.Log(printables...)
	t.Helper()
	FailImpl(t, printables...)
}

var Red = "\033[31;1m"
var Clear = "\033[0;0m"

func FailImpl(t *testing.T, printables ...interface{}) {
	t.Helper()
	t.Fatal(Red, printables, Clear)
}
