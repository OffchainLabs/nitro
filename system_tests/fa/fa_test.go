package fa_test

import (
	"errors"
	"github.com/offchainlabs/nitro/util/testhelpers"
	"testing"
)

func TestAliasing(t *testing.T) {
	var err error
	err = errors.New("test")
	Require(t, err)
}

func Require(t *testing.T, err error, text ...interface{}) {
	t.Helper()
	testhelpers.RequireImpl(t, err, text...)
}
