//
// Copyright 2022, Offchain Labs, Inc. All rights reserved.
//

package das

import (
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbstate"
)

func TestDAS_BasicAggregationLocal(t *testing.T) {
	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(false)))
	glogger.Verbosity(log.LvlTrace)
	log.Root().SetHandler(glogger)

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

	aggregator := NewAggregator(AggregatorConfig{1}, backends)
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

type FailsStore struct {
	DataAvailabilityService
}

type FailsRetrieve struct {
	DataAvailabilityService
}

func (*FailsRetrieve) Retrieve(ctx context.Context, cert []byte) ([]byte, error) {
	return nil, errors.New("Expected Retrieve failure")
}

func (*FailsStore) Store(ctx context.Context, message []byte) (*arbstate.DataAvailabilityCertificate, error) {
	return nil, errors.New("Expected Store failure")
}

func TestDAS_LessThanHStorageFailures(t *testing.T) {
	numBackendDAS := 10
	var backends []serviceDetails
	for i := 0; i < numBackendDAS; i++ {
		dbPath, err := ioutil.TempDir("/tmp", "das_test")
		Require(t, err)
		defer os.RemoveAll(dbPath)

		das, err := NewLocalDiskDataAvailabilityService(dbPath, 1<<i)
		Require(t, err)

		details := serviceDetails{das, *das.pubKey}
		if i < 3 {
			details.service = &FailsStore{das}
		}

		backends = append(backends, details)
	}

	aggregator := NewAggregator(AggregatorConfig{4}, backends)
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

func TestDAS_HStorageFailures(t *testing.T) {
	numBackendDAS := 10
	var backends []serviceDetails
	for i := 0; i < numBackendDAS; i++ {
		dbPath, err := ioutil.TempDir("/tmp", "das_test")
		Require(t, err)
		defer os.RemoveAll(dbPath)

		das, err := NewLocalDiskDataAvailabilityService(dbPath, 1<<i)
		Require(t, err)

		details := serviceDetails{das, *das.pubKey}
		if i < 3 {
			details.service = &FailsStore{das}
		}

		backends = append(backends, details)
	}

	aggregator := NewAggregator(AggregatorConfig{3}, backends)
	ctx := context.Background()

	rawMsg := []byte("It's time for you to see the fnords.")
	_, err := aggregator.Store(ctx, rawMsg)
	if err == nil {
		Fail(t, "Expected error from too many failed DASes.")
	}
}
