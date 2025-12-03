// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package das

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/blsSignatures"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/daprovider/das/dasutil"
	"github.com/offchainlabs/nitro/util/testhelpers/flag"
)

func TestDAS_BasicAggregationLocal(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	numBackendDAS := 10
	var backends []ServiceDetails
	var storageServices []StorageService
	for i := 0; i < numBackendDAS; i++ {
		privKey, err := blsSignatures.GeneratePrivKeyString()
		Require(t, err)

		config := DefaultDataAvailabilityConfig
		config.Enable = true
		config.Key.PrivKey = privKey

		storageServices = append(storageServices, NewMemoryBackedStorageService(ctx))
		das, err := NewSignAfterStoreDASWriter(ctx, config, storageServices[i])
		Require(t, err)
		signerMask := uint64(1 << i)
		details, err := NewServiceDetails(das, *das.pubKey, signerMask, "service"+strconv.Itoa(i))
		Require(t, err)
		backends = append(backends, *details)
	}

	aggregatorConfig := DefaultAggregatorConfig
	aggregatorConfig.AssumedHonest = 1
	daConfig := DefaultDataAvailabilityConfig
	daConfig.RPCAggregator = aggregatorConfig
	aggregator, err := newAggregator(daConfig, backends)
	Require(t, err)

	rawMsg := []byte("It's time for you to see the fnords.")
	cert, err := aggregator.Store(ctx, rawMsg, 0)
	Require(t, err, "Error storing message")

	for _, storageService := range storageServices {
		messageRetrieved, err := storageService.GetByHash(ctx, cert.DataHash)
		Require(t, err, "Failed to retrieve message")
		if !bytes.Equal(rawMsg, messageRetrieved) {
			Fail(t, "Retrieved message is not the same as stored one.")
		}
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
	dasutil.DASWriter
}

func (w *WrapStore) Store(ctx context.Context, message []byte, timeout uint64) (*dasutil.DataAvailabilityCertificate, error) {
	switch w.injector.shouldFail() {
	case success:
		return w.DASWriter.Store(ctx, message, timeout)
	case immediateError:
		return nil, errors.New("expected Store failure")
	case tooSlow:
		<-ctx.Done()
		return nil, ctx.Err()
	case dataCorruption:
		cert, err := w.DASWriter.Store(ctx, message, timeout)
		if err != nil {
			return nil, err
		}
		cert.DataHash[0] = ^cert.DataHash[0]
		return cert, nil
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
	glogger := log.NewGlogHandler(
		log.NewTerminalHandler(io.Writer(os.Stderr), false))
	glogger.Verbosity(log.LevelTrace)
	log.SetDefault(log.NewLogger(glogger))
}

func testConfigurableStorageFailures(t *testing.T, shouldFailAggregation bool) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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

	injectedFailures := newRandomBagOfFailures(t, nSuccesses, nFailures, dataCorruption)
	var backends []ServiceDetails
	var storageServices []StorageService
	for i := 0; i < numBackendDAS; i++ {
		privKey, err := blsSignatures.GeneratePrivKeyString()
		Require(t, err)

		config := DefaultDataAvailabilityConfig
		config.Enable = true
		config.Key.PrivKey = privKey

		storageServices = append(storageServices, NewMemoryBackedStorageService(ctx))
		das, err := NewSignAfterStoreDASWriter(ctx, config, storageServices[i])
		Require(t, err)
		signerMask := uint64(1 << i)
		details, err := NewServiceDetails(&WrapStore{t, injectedFailures, das}, *das.pubKey, signerMask, "service"+strconv.Itoa(i))
		Require(t, err)
		backends = append(backends, *details)
	}

	aggregatorConfig := DefaultAggregatorConfig
	aggregatorConfig.AssumedHonest = assumedHonest
	daConfig := DefaultDataAvailabilityConfig
	daConfig.RPCAggregator = aggregatorConfig
	daConfig.RequestTimeout = time.Millisecond * 2000
	aggregator, err := newAggregator(daConfig, backends)
	Require(t, err)

	rawMsg := []byte("It's time for you to see the fnords.")
	cert, err := aggregator.Store(ctx, rawMsg, 0)
	if !shouldFailAggregation {
		Require(t, err, "Error storing message")
	} else {
		if err == nil {
			Fail(t, "Expected error from too many failed DASes.")
		}
		return
	}

	// Wait for all stores that would succeed to succeed.
	time.Sleep(time.Millisecond * 2000)
	retrievalFailures := 0
	for _, storageService := range storageServices {
		messageRetrieved, err := storageService.GetByHash(ctx, cert.DataHash)
		if err != nil {
			retrievalFailures++
		} else if !bytes.Equal(rawMsg, messageRetrieved) {
			retrievalFailures++
		}
	}
	if retrievalFailures > nFailures {
		Fail(t, fmt.Sprintf("retrievalFailures(%d) > nFailures(%d)", retrievalFailures, nFailures))
	}
}

func initTest(t *testing.T) int {
	t.Parallel()
	seed := time.Now().UnixNano()
	if len(*testflag.SeedFlag) > 0 {
		var err error
		intSeed, err := strconv.Atoi(*testflag.SeedFlag)
		Require(t, err, "Failed to parse string")
		seed = int64(intSeed)
	}
	rand.Seed(seed)

	runs := 2 ^ 32
	if len(*testflag.RunsFlag) > 0 {
		var err error
		runs, err = strconv.Atoi(*testflag.RunsFlag)
		Require(t, err, "Failed to parse string")
	}

	if len(*testflag.LoggingFlag) > 0 {
		enableLogging()
	}

	log.Trace(fmt.Sprintf("Running test with seed %d", seed))

	return runs
}

func TestDAS_LessThanHStorageFailures(t *testing.T) {
	runs := initTest(t)

	for i := 0; i < min(runs, 20); i++ {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			t.Parallel()
			testConfigurableStorageFailures(t, false)
		})
	}
}

func TestDAS_AtLeastHStorageFailures(t *testing.T) {
	runs := initTest(t)
	for i := 0; i < min(runs, 10); i++ {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			t.Parallel()
			testConfigurableStorageFailures(t, true)
		})
	}
}

func TestDAS_InsufficientBackendsTriggersFallback(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up aggregator with 5 backends, AssumedHonest=2
	// This means we need K=5+1-2=4 successful responses to succeed
	// We'll inject exactly 2 failures, causing aggregation to fail
	numBackendDAS := 5
	assumedHonest := 2
	nFailures := 2
	nSuccesses := numBackendDAS - nFailures

	log.Trace(fmt.Sprintf("Testing aggregator fallback with K:%d (K=N+1-H), N:%d, H:%d, and %d successes", numBackendDAS+1-assumedHonest, numBackendDAS, assumedHonest, nSuccesses))

	injectedFailures := newRandomBagOfFailures(t, nSuccesses, nFailures, immediateError)
	var backends []ServiceDetails
	for i := 0; i < numBackendDAS; i++ {
		privKey, err := blsSignatures.GeneratePrivKeyString()
		Require(t, err)

		config := DefaultDataAvailabilityConfig
		config.Enable = true
		config.Key.PrivKey = privKey

		storageService := NewMemoryBackedStorageService(ctx)
		das, err := NewSignAfterStoreDASWriter(ctx, config, storageService)
		Require(t, err)
		signerMask := uint64(1 << i)
		details, err := NewServiceDetails(&WrapStore{t, injectedFailures, das}, *das.pubKey, signerMask, "service"+strconv.Itoa(i))
		Require(t, err)
		backends = append(backends, *details)
	}

	aggregatorConfig := DefaultAggregatorConfig
	aggregatorConfig.AssumedHonest = assumedHonest
	daConfig := DefaultDataAvailabilityConfig
	daConfig.RPCAggregator = aggregatorConfig
	daConfig.RequestTimeout = time.Millisecond * 2000
	aggregator, err := newAggregator(daConfig, backends)
	Require(t, err)

	// Wrap the aggregator with writerForDAS to test error conversion
	// Use 0 for maxMessageSize to indicate use default
	writer := dasutil.NewWriterForDAS(aggregator, 0)

	rawMsg := []byte("It's time for you to see the fnords.")
	promise := writer.Store(rawMsg, 0)

	// Wait for the promise to complete
	result, err := promise.Await(ctx)
	if err == nil {
		Fail(t, "Expected error from insufficient backends, got nil")
	}

	// Verify the error contains ErrFallbackRequested
	if !errors.Is(err, daprovider.ErrFallbackRequested) {
		Fail(t, fmt.Sprintf("Expected error to contain ErrFallbackRequested, got: %v", err))
	}

	// Also verify the original error is preserved
	if !errors.Is(err, dasutil.ErrBatchToDasFailed) {
		Fail(t, fmt.Sprintf("Expected error to contain ErrBatchToDasFailed, got: %v", err))
	}

	log.Info("Fallback error correctly returned", "error", err, "result", result)
}
