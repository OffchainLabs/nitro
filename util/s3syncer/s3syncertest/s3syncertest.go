// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package s3syncertest

import (
	"bytes"
	"net/http/httptest"
	"testing"

	"github.com/johannesboyne/gofakes3"
	"github.com/johannesboyne/gofakes3/backend/s3mem"
)

func NewFakeS3(t *testing.T, bucket string, objects map[string][]byte) (string, gofakes3.Backend) {
	t.Helper()
	backend := s3mem.New()
	if err := backend.CreateBucket(bucket); err != nil {
		t.Fatalf("CreateBucket: %v", err)
	}
	for key, body := range objects {
		if _, err := backend.PutObject(bucket, key, map[string]string{}, bytes.NewReader(body), int64(len(body)), &gofakes3.PutConditions{}); err != nil {
			t.Fatalf("PutObject %q: %v", key, err)
		}
	}
	server := httptest.NewServer(gofakes3.New(backend).Server())
	t.Cleanup(server.Close)
	return server.URL, backend
}
