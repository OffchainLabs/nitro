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
	numBackendDAS := 10
	var backends []serviceDetails
	for i := 0; i < numBackendDAS; i++ {
		dbPath, err := ioutil.TempDir("/tmp", "das_test")
		Require(t, err)
		defer os.RemoveAll(dbPath)

		signerMask := uint64(1 << i)
		das, err := NewLocalDiskDataAvailabilityService(dbPath, signerMask)
		Require(t, err)
		backends = append(backends, serviceDetails{das, *das.pubKey, signerMask})
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
	tooSlow
	dataCorruption
)

type failureInjector interface {
	shouldFail() failureType
}

type randomBagOfFailures struct {
	t        *testing.T
	failures []failureType
	mutex    sync.Mutex
}

func newRandomBagOfFailures(t *testing.T, nSuccess, nFailures int, highestFailureType failureType) *randomBagOfFailures {
	var failures []failureType
	for i := 0; i < nSuccess; i++ {
		failures = append(failures, success)
	}

	rand.Seed(time.Now().UnixNano())
	for i := 0; i < nFailures; i++ {
		failures = append(failures, failureType(rand.Int()%int(highestFailureType)+1))
	}

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
	case tooSlow:
		<-ctx.Done()
		return nil, errors.New("Canceled")
	case dataCorruption:
		data, err := w.DataAvailabilityService.Retrieve(ctx, cert)
		if err != nil {
			return nil, err
		}
		data[0] = ^data[0]
		return data, nil
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
	case tooSlow:
		<-ctx.Done()
		return nil, errors.New("Canceled")
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

func enableLogging() {
	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(false)))
	glogger.Verbosity(log.LvlTrace)
	log.Root().SetHandler(glogger)
}

func testConfigurableStorageFailures(t *testing.T, shouldFailAggregation bool) {
	rand.Seed(time.Now().UnixNano())
	numBackendDAS := (rand.Int() % 20) + 1
	assumedHonest := (rand.Int() % numBackendDAS) + 1
	var nFailures int
	if shouldFailAggregation {
		nFailures = max(assumedHonest, rand.Int()%(numBackendDAS+1))
	} else {
		nFailures = min(assumedHonest-1, rand.Int()%(numBackendDAS+1))
	}
	nSuccesses := numBackendDAS - nFailures
	log.Trace(fmt.Sprintf("Testing aggregator with K:%d with K=N+1-H, N:%d, H:%d, and %d successes", numBackendDAS+1-assumedHonest, numBackendDAS, assumedHonest, nSuccesses))

	injectedFailures := newRandomBagOfFailures(t, nSuccesses, nFailures, tooSlow)
	var backends []serviceDetails
	for i := 0; i < numBackendDAS; i++ {
		dbPath, err := ioutil.TempDir("/tmp", "das_test")
		Require(t, err)
		defer os.RemoveAll(dbPath)

		signerMask := uint64(1 << i)
		das, err := NewLocalDiskDataAvailabilityService(dbPath, signerMask)
		Require(t, err)

		details := serviceDetails{&WrapStore{t, injectedFailures, das}, *das.pubKey, signerMask}

		backends = append(backends, details)
	}

	aggregator := DeadlineWrapper{time.Millisecond * 500, NewAggregator(AggregatorConfig{assumedHonest, 7 * 24 * time.Hour}, backends)}
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
	for i := 0; i < 10; i++ {
		testConfigurableStorageFailures(t, true)
	}
}

func testConfigurableRetrieveFailures(t *testing.T, shouldFail bool) {
	rand.Seed(time.Now().UnixNano())
	numBackendDAS := (rand.Int() % 20) + 1
	var nSuccesses, nFailures int
	if shouldFail {
		nSuccesses = 0
		nFailures = numBackendDAS
	} else {
		nSuccesses = (rand.Int() % numBackendDAS) + 1
		nFailures = numBackendDAS - nSuccesses
	}

	var backends []serviceDetails
	injectedFailures := newRandomBagOfFailures(t, nSuccesses, nFailures, dataCorruption)
	for i := 0; i < numBackendDAS; i++ {
		dbPath, err := ioutil.TempDir("/tmp", "das_test")
		Require(t, err)
		defer os.RemoveAll(dbPath)

		signerMask := uint64(1 << i)
		das, err := NewLocalDiskDataAvailabilityService(dbPath, signerMask)
		Require(t, err)

		details := serviceDetails{&WrapRetrieve{t, injectedFailures, das}, *das.pubKey, signerMask}

		backends = append(backends, details)
	}

	aggregator := DeadlineWrapper{time.Millisecond * 500, NewAggregator(AggregatorConfig{1, 7 * 24 * time.Hour}, backends)}
	ctx := context.Background()

	rawMsg := []byte("It's time for you to see the fnords.")
	cert, err := aggregator.Store(ctx, rawMsg, CALLEE_PICKS_TIMEOUT)
	Require(t, err, "Error storing message")

	messageRetrieved, err := aggregator.Retrieve(ctx, Serialize(*cert))
	if !shouldFail {
		Require(t, err, "Error retrieving message")
	} else {
		if err == nil {
			Fail(t, "Expected error from too many failed DASes.")
		}
		return
	}
	if !bytes.Equal(rawMsg, messageRetrieved) {
		Fail(t, "Retrieved message is not the same as stored one.")
	}
}

func TestDAS_RetrieveFailureFromSomeDASes(t *testing.T) {
	for i := 0; i < 20; i++ {
		testConfigurableRetrieveFailures(t, false)
	}
}

func TestDAS_RetrieveFailureFromAllDASes(t *testing.T) {
	for i := 0; i < 10; i++ {
		testConfigurableRetrieveFailures(t, true)
	}
}
