// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package forwarder

import (
	"testing"
	"time"

	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/util/sqsclient"
)

func NewTestForwarder(t *testing.T, queueClient sqsclient.QueueClient, endpointURL string) *Forwarder {
	t.Helper()
	config := &Config{
		Workers:            1,
		PollInterval:       time.Second,
		SQSWaitTimeSeconds: DefaultConfig.SQSWaitTimeSeconds,
		ExternalEndpoint: genericconf.HTTPClientConfig{
			URL:     endpointURL,
			Timeout: genericconf.HTTPClientConfigDefault.Timeout,
		},
	}
	fwd, err := New(config, queueClient)
	if err != nil {
		t.Fatal(err)
	}
	return fwd
}
