package fa_test

import (
	"github.com/offchainlabs/nitro/util/testhelpers"
	"testing"
)

func TestAliasing(t *testing.T) {
	Fail(t, "fail")
}

func Fail(t *testing.T, printables ...interface{}) {
	t.Helper()
	testhelpers.FailImpl(t, printables...)
}
