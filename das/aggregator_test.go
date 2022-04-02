//
// Copyright 2022, Offchain Labs, Inc. All rights reserved.
//

package das

import (
	"context"
	"io/ioutil"
	"os"
	"testing"
)

func TestDASAggregationLocal(t *testing.T) {
	numBackendDAS := 10
	var backends []serviceDetails
	for i := 0; i < numBackendDAS; i++ {
		dbPath, err := ioutil.TempDir("/tmp", "das_test")
		Require(t, err)
		defer os.RemoveAll(dbPath)

		das, err := NewLocalDiskDataAvailabilityService(dbPath, 1<<i)
		Require(t, err)
		backends = append(backends, serviceDetails{das, *das.pubKey})
	}

	aggregator := NewAggregator(backends)
	ctx := context.Background()

	rawMsg := []byte("It's time for you to see the fnords.")
	_, err := aggregator.Store(ctx, rawMsg)
	Require(t, err, "Error storing message")
}
