// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package anytrust

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

type restfulClientContextCall struct {
	name string
	call func(context.Context, *RestfulClient) error
}

func restfulClientContextCalls() []restfulClientContextCall {
	return []restfulClientContextCall{
		{
			name: "HealthCheck",
			call: func(ctx context.Context, client *RestfulClient) error {
				return client.HealthCheck(ctx)
			},
		},
		{
			name: "ExpirationPolicy",
			call: func(ctx context.Context, client *RestfulClient) error {
				_, err := client.ExpirationPolicy(ctx)
				return err
			},
		},
	}
}

func TestRestfulClientCanceledContext(t *testing.T) {
	for _, test := range restfulClientContextCalls() {
		t.Run(test.name, func(t *testing.T) {
			requestReceived := make(chan struct{}, 1)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				requestReceived <- struct{}{}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"expirationPolicy":"KeepForever"}`))
			}))
			defer server.Close()

			client, err := NewRestfulClientFromURL(server.URL)
			if err != nil {
				t.Fatalf("failed to create REST client: %v", err)
			}

			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			start := time.Now()
			err = test.call(ctx, client)
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("expected context.Canceled, got %v", err)
			}
			if elapsed := time.Since(start); elapsed > time.Second {
				t.Fatalf("request took %v to return after cancellation", elapsed)
			}
			select {
			case <-requestReceived:
				t.Fatal("request reached the server despite the canceled context")
			default:
			}
		})
	}
}

func TestRestfulClientDeadline(t *testing.T) {
	for _, test := range restfulClientContextCalls() {
		t.Run(test.name, func(t *testing.T) {
			releaseHandler := make(chan struct{})
			var releaseOnce sync.Once
			release := func() {
				releaseOnce.Do(func() {
					close(releaseHandler)
				})
			}

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				select {
				case <-r.Context().Done():
					return
				case <-releaseHandler:
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"expirationPolicy":"KeepForever"}`))
				}
			}))
			defer func() {
				release()
				server.Close()
			}()

			client, err := NewRestfulClientFromURL(server.URL)
			if err != nil {
				t.Fatalf("failed to create REST client: %v", err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			result := make(chan error, 1)
			start := time.Now()
			go func() {
				result <- test.call(ctx, client)
			}()

			select {
			case err := <-result:
				if !errors.Is(err, context.DeadlineExceeded) {
					t.Fatalf("expected context.DeadlineExceeded, got %v", err)
				}
				if elapsed := time.Since(start); elapsed > time.Second {
					t.Fatalf("request took %v to return after its deadline", elapsed)
				}
			case <-time.After(2 * time.Second):
				release()
				err := <-result
				t.Fatalf("request did not return after its context deadline; returned %v after the handler was released", err)
			}
		})
	}
}
