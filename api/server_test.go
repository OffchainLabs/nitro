package api_test

import (
	"testing"

	"github.com/OffchainLabs/bold/api"
)

func NewTestServer(t *testing.T) (*api.Server, *FakeEdgesProvider, *FakeAssertionProvider) {
	t.Helper()

	edges := &FakeEdgesProvider{}
	assertions := &FakeAssertionProvider{}

	s, err := api.NewServer(&api.Config{
		EdgesProvider:      edges,
		AssertionsProvider: assertions,
	})
	if err != nil {
		t.Fatal(err)
	}

	if !s.Registered() {
		t.Fatal("server methods not registered")
	}

	return s, edges, assertions
}
