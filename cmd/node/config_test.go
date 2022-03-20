package main

import (
	"context"
	"github.com/offchainlabs/nitro/util/testhelpers"
	"testing"
)

func TestConfig(t *testing.T) {
	_, _, _, err := ParseNode(context.Background())
	testhelpers.RequireImpl(t, err)
}
