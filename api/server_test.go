package api_test

import (
	"testing"

	"github.com/OffchainLabs/bold/api"
)

func NewTestServer(t *testing.T) (*api.Server, *FakeDataAccessor) {
	t.Helper()

	data := &FakeDataAccessor{}

	s, err := api.NewServer(&api.Config{
		DataAccessor: data,
	})
	if err != nil {
		t.Fatal(err)
	}

	if !s.Registered() {
		t.Fatal("server methods not registered")
	}

	return s, data
}
