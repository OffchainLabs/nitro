//
// Copyright 2022, Offchain Labs, Inc. All rights reserved.
//

package das

import (
	"bytes"
	"context"
	"io/ioutil"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/log"
)

func TestDASBasicAggregationLocal(t *testing.T) {
	/*
		glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(false)))
		glogger.Verbosity(log.LvlTrace)
		log.Root().SetHandler(glogger)
	*/
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
	cert, err := aggregator.Store(ctx, rawMsg)
	Require(t, err, "Error storing message")

	messageRetrieved, err := aggregator.Retrieve(ctx, Serialize(*cert))
	Require(t, err, "Failed to retrieve message")
	if !bytes.Equal(rawMsg, messageRetrieved) {
		Fail(t, "Retrieved message is not the same as stored one.")
	}
}
