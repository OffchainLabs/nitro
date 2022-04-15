//
// Copyright 2022, Offchain Labs, Inc. All rights reserved.
//

package das

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"sync"
	"testing"
	"time"

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

	aggregator := NewAggregator(AggregatorConfig{1, 7 * 24 * time.Hour}, backends)
	ctx := context.Background()

	rawMsg := []byte("It's time for you to see the fnords.")
	cert, err := aggregator.Store(ctx, rawMsg, CALLEE_PICKS_TIMEOUT)
	Require(t, err, "Error storing message")

	messageRetrieved, err := aggregator.Retrieve(ctx, Serialize(*cert))
	Require(t, err, "Failed to retrieve message")
	if !bytes.Equal(rawMsg, messageRetrieved) {
		Fail(t, "Retrieved message is not the same as stored one.")
	}
}

type failureType int

const (
	success failureType = iota
	immediateError
	// TODO timeoutError
)

type failureInjector interface {
	shouldFail() failureType
}

type randomBagOfFailures struct {
	t        *testing.T
	failures []failureType
	mutex    sync.Mutex
}

func newRandomBagOfFailures(t *testing.T, nSuccess, nImmediateError int) *randomBagOfFailures {
	var failures []failureType
	for i := 0; i < nSuccess; i++ {
		failures = append(failures, success)
	}
	for i := 0; i < nImmediateError; i++ {
		failures = append(failures, immediateError)
	}

	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(failures), func(i, j int) { failures[i], failures[j] = failures[j], failures[i] })

	log.Trace("Injected failures", "failures", failures)

	return &randomBagOfFailures{
		t:        t,
		failures: failures,
	}
}

func (b *randomBagOfFailures) shouldFail() failureType {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	if len(b.failures) == 0 {
		Fail(b.t, "shouldFail called more times than expected")
	}

	toReturn := b.failures[0]
	b.failures = b.failures[1:]
	return toReturn
}

type WrapStore struct {
	t        *testing.T
	injector failureInjector
	DataAvailabilityService
}

type WrapRetrieve struct {
	t        *testing.T
	injector failureInjector
	DataAvailabilityService
}

func (w *WrapRetrieve) Retrieve(ctx context.Context, cert []byte) ([]byte, error) {
	switch w.injector.shouldFail() {
	case success:
		return w.DataAvailabilityService.Retrieve(ctx, cert)
	case immediateError:
		return nil, errors.New("Expected Retrieve failure")
	}

	Fail(w.t)
	return nil, nil
}

func (w *WrapStore) Store(ctx context.Context, message []byte, timeout uint64) (*arbstate.DataAvailabilityCertificate, error) {
	switch w.injector.shouldFail() {
	case success:
		return w.DataAvailabilityService.Store(ctx, message, timeout)
	case immediateError:
		return nil, errors.New("Expected Store failure")
	}

	Fail(w.t)
	return nil, nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func testConfigurableStorageFailures(t *testing.T, shouldFailAggregation bool) {
	/*
		glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(false)))
		glogger.Verbosity(log.LvlTrace)
		log.Root().SetHandler(glogger)
	*/
	rand.Seed(time.Now().UnixNano())
	numBackendDAS := (rand.Int() % 20) + 1
	assumedHonest := (rand.Int() % numBackendDAS) + 1
	var nImmediateErrors int
	if shouldFailAggregation {
		nImmediateErrors = max(assumedHonest, rand.Int()%(numBackendDAS+1))
	} else {
		nImmediateErrors = min(assumedHonest-1, rand.Int()%(numBackendDAS+1))
	}
	nSuccesses := numBackendDAS - nImmediateErrors
	log.Trace(fmt.Sprintf("Testing aggregator with K:%d with K=N+1-H, N:%d, H:%d, and %d successes", numBackendDAS+1-assumedHonest, numBackendDAS, assumedHonest, nSuccesses))

	injectedFailures := newRandomBagOfFailures(t, nSuccesses, nImmediateErrors)
	var backends []serviceDetails
	for i := 0; i < numBackendDAS; i++ {
		dbPath, err := ioutil.TempDir("/tmp", "das_test")
		Require(t, err)
		defer os.RemoveAll(dbPath)

		das, err := NewLocalDiskDataAvailabilityService(dbPath, 1<<i)
		Require(t, err)

		details := serviceDetails{&WrapStore{t, injectedFailures, das}, *das.pubKey}

		backends = append(backends, details)
	}

	aggregator := NewAggregator(AggregatorConfig{assumedHonest, 7 * 24 * time.Hour}, backends)
	ctx := context.Background()

	rawMsg := []byte("It's time for you to see the fnords.")
	cert, err := aggregator.Store(ctx, rawMsg, CALLEE_PICKS_TIMEOUT)
	if !shouldFailAggregation {
		Require(t, err, "Error storing message")
	} else {
		if err == nil {
			Fail(t, "Expected error from too many failed DASes.")
		}
		return
	}

	messageRetrieved, err := aggregator.Retrieve(ctx, Serialize(*cert))
	Require(t, err, "Failed to retrieve message")
	if !bytes.Equal(rawMsg, messageRetrieved) {
		Fail(t, "Retrieved message is not the same as stored one.")
	}

}

func TestDAS_LessThanHStorageFailures(t *testing.T) {
	for i := 0; i < 100; i++ {
		testConfigurableStorageFailures(t, false)
	}
}

func TestDAS_AtLeastHStorageFailures(t *testing.T) {
	for i := 0; i < 100; i++ {
		testConfigurableStorageFailures(t, true)
	}
}
