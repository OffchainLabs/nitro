package main

import (
	"context"
	"github.com/offchainlabs/nitro/util/testhelpers"
	"testing"
)

func TestConfig(t *testing.T) {
	_, err := ParseRelay(context.Background(), []string{})
	testhelpers.RequireImpl(t, err)
}
