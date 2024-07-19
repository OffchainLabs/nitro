package N

import (
	"errors"
	"github.com/offchainlabs/nitro/util/testhelpers"
	"testing"
)

func TestEmptyCliConfig(t *testing.T) {
	err := errors.New("please retry")
	Require(t, err)
}

func Require(t *testing.T, err error, text ...interface{}) {
	t.Helper()
	testhelpers.RequireImpl(t, err, text...)
}
