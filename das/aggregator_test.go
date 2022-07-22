// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbstate"
)

func TestDAS_BasicAggregationLocal(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	numBackendDAS := 10
	var backends []ServiceDetails
	for i := 0; i < numBackendDAS; i++ {
		dbPath := t.TempDir()
		_, _, err := GenerateAndStoreKeys(dbPath)
		Require(t, err)

		config := DataAvailabilityConfig{
			Enable: true,
			KeyConfig: KeyConfig{
				KeyDir: dbPath,
			},
			LocalFileStorageConfig: LocalFileStorageConfig{
				Enable:  true,
				DataDir: dbPath,
			},
			L1NodeURL: "none",
		}

		storageService, lifecycleManager, err := CreatePersistentStorageService(ctx, &config)
		Require(t, err)
		defer lifecycleManager.StopAndWaitUntil(time.Second)
		das, err := NewSignAfterStoreDAS(ctx, config, storageService)
		Require(t, err)
		pubKey, _, err := ReadKeysFromFile(dbPath)
		Require(t, err)
		signerMask := uint64(1 << i)
		details, err := NewServiceDetails(das, *pubKey, signerMask)
		Require(t, err)
		backends = append(backends, *details)
	}

	aggregator, err := NewAggregator(ctx, DataAvailabilityConfig{AggregatorConfig: AggregatorConfig{AssumedHonest: 1}, L1NodeURL: "none"}, backends)
	Require(t, err)

	rawMsg := []byte("It's time for you to see the fnords.")
	cert, err := aggregator.Store(ctx, rawMsg, 0, []byte{})
	Require(t, err, "Error storing message")

	messageRetrieved, err := aggregator.GetByHash(ctx, cert.DataHash)
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

type WrapGetByHash struct {
	t        *testing.T
	injector failureInjector
	DataAvailabilityService
}

func (w *WrapGetByHash) GetByHash(ctx context.Context, hash common.Hash) ([]byte, error) {
	switch w.injector.shouldFail() {
	case success:
		return w.DataAvailabilityService.GetByHash(ctx, hash)
	case immediateError:
		return nil, errors.New("Expected Retrieve failure")
	case tooSlow:
		<-ctx.Done()
		return nil, errors.New("Canceled")
	case dataCorruption:
		data, err := w.DataAvailabilityService.GetByHash(ctx, hash)
		if err != nil {
			return nil, err
		}
		data[0] = ^data[0]
		return data, nil
	}
	Fail(w.t)
	return nil, nil
}

func (w *WrapStore) Store(ctx context.Context, message []byte, timeout uint64, sig []byte) (*arbstate.DataAvailabilityCertificate, error) {
	switch w.injector.shouldFail() {
	case success:
		return w.DataAvailabilityService.Store(ctx, message, timeout, sig)
	case immediateError:
		return nil, errors.New("Expected Store failure")
	case tooSlow:
		<-ctx.Done()
		return nil, errors.New("Canceled")
	case dataCorruption:
		cert, err := w.DataAvailabilityService.Store(ctx, message, timeout, sig)
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
	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(false)))
	glogger.Verbosity(log.LvlTrace)
	log.Root().SetHandler(glogger)
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
	for i := 0; i < numBackendDAS; i++ {
		dbPath := t.TempDir()
		_, _, err := GenerateAndStoreKeys(dbPath)
		Require(t, err)

		config := DataAvailabilityConfig{
			Enable: true,
			KeyConfig: KeyConfig{
				KeyDir: dbPath,
			},
			LocalFileStorageConfig: LocalFileStorageConfig{
				Enable:  true,
				DataDir: dbPath,
			},
			L1NodeURL: "none",
		}

		storageService, lifecycleManager, err := CreatePersistentStorageService(ctx, &config)
		Require(t, err)
		defer lifecycleManager.StopAndWaitUntil(time.Second)
		das, err := NewSignAfterStoreDAS(ctx, config, storageService)
		Require(t, err)
		pubKey, _, err := ReadKeysFromFile(dbPath)
		Require(t, err)
		signerMask := uint64(1 << i)
		details, err := NewServiceDetails(&WrapStore{t, injectedFailures, das}, *pubKey, signerMask)
		Require(t, err)
		backends = append(backends, *details)
	}

	unwrappedAggregator, err := NewAggregator(ctx, DataAvailabilityConfig{AggregatorConfig: AggregatorConfig{AssumedHonest: assumedHonest}, L1NodeURL: "none"}, backends)
	Require(t, err)
	aggregator := TimeoutWrapper{time.Millisecond * 2000, unwrappedAggregator}

	rawMsg := []byte("It's time for you to see the fnords.")
	cert, err := aggregator.Store(ctx, rawMsg, 0, []byte{})
	if !shouldFailAggregation {
		Require(t, err, "Error storing message")
	} else {
		if err == nil {
			Fail(t, "Expected error from too many failed DASes.")
		}
		return
	}

	messageRetrieved, err := aggregator.GetByHash(ctx, cert.DataHash)
	Require(t, err, "Failed to retrieve message")
	if !bytes.Equal(rawMsg, messageRetrieved) {
		Fail(t, "Retrieved message is not the same as stored one.")
	}
}

func initTest(t *testing.T) int {
	t.Parallel()
	seed := time.Now().UnixNano()
	seedStr := os.Getenv("SEED")
	if len(seedStr) > 0 {
		var err error
		intSeed, err := strconv.Atoi(seedStr)
		Require(t, err, "Failed to parse string")
		seed = int64(intSeed)
	}
	rand.Seed(seed)

	log.Trace(fmt.Sprintf("Running test with seed %d", seed))

	runsStr := os.Getenv("RUNS")
	runs := 2 ^ 32
	if len(runsStr) > 0 {
		var err error
		runs, err = strconv.Atoi(runsStr)
		Require(t, err, "Failed to parse string")
	}

	loggingStr := os.Getenv("LOGGING")
	if len(loggingStr) > 0 {
		enableLogging()
	}

	return runs
}

func TestDAS_LessThanHStorageFailures(t *testing.T) {
	runs := initTest(t)

	for i := 0; i < min(runs, 20); i++ {
		testConfigurableStorageFailures(t, false)
	}
}

func TestDAS_AtLeastHStorageFailures(t *testing.T) {
	runs := initTest(t)

	for i := 0; i < min(runs, 10); i++ {
		testConfigurableStorageFailures(t, true)
	}
}

func testConfigurableRetrieveFailures(t *testing.T, shouldFail bool) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	numBackendDAS := (rand.Int() % 20) + 1
	var nSuccesses, nFailures int
	if shouldFail {
		nSuccesses = 0
		nFailures = numBackendDAS
	} else {
		nSuccesses = (rand.Int() % numBackendDAS) + 1
		nFailures = numBackendDAS - nSuccesses
	}
	log.Trace(fmt.Sprintf("Testing aggregator retrieve with %d successes and %d failures", nSuccesses, nFailures))
	var backends []ServiceDetails
	injectedFailures := newRandomBagOfFailures(t, nSuccesses, nFailures, dataCorruption)
	for i := 0; i < numBackendDAS; i++ {
		dbPath := t.TempDir()
		_, _, err := GenerateAndStoreKeys(dbPath)
		Require(t, err)

		config := DataAvailabilityConfig{
			Enable: true,
			KeyConfig: KeyConfig{
				KeyDir: dbPath,
			},
			LocalFileStorageConfig: LocalFileStorageConfig{
				Enable:  true,
				DataDir: dbPath,
			},
			L1NodeURL: "none",
		}

		storageService, lifecycleManager, err := CreatePersistentStorageService(ctx, &config)
		Require(t, err)
		defer lifecycleManager.StopAndWaitUntil(time.Second)
		das, err := NewSignAfterStoreDAS(ctx, config, storageService)
		Require(t, err)
		pubKey, _, err := ReadKeysFromFile(dbPath)
		Require(t, err)
		signerMask := uint64(1 << i)
		details := ServiceDetails{&WrapGetByHash{t, injectedFailures, das}, *pubKey, signerMask}

		backends = append(backends, details)
	}

	// All honest -> at least 1 store succeeds.
	// Aggregator should collect responses up until end of deadline, so
	// it should get all successes.
	unwrappedAggregator, err := NewAggregator(ctx, DataAvailabilityConfig{AggregatorConfig: AggregatorConfig{AssumedHonest: numBackendDAS}, L1NodeURL: "none"}, backends)
	Require(t, err)
	aggregator := TimeoutWrapper{time.Millisecond * 2000, unwrappedAggregator}

	rawMsg := []byte("It's time for you to see the fnords.")
	cert, err := aggregator.Store(ctx, rawMsg, 0, []byte{})
	Require(t, err, "Error storing message")

	messageRetrieved, err := aggregator.GetByHash(ctx, cert.DataHash)
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
	runs := initTest(t)
	for i := 0; i < min(runs, 10); i++ {
		testConfigurableRetrieveFailures(t, false)
	}
}

func TestDAS_RetrieveFailureFromAllDASes(t *testing.T) {
	runs := initTest(t)
	for i := 0; i < min(runs, 10); i++ {
		testConfigurableRetrieveFailures(t, true)
	}
}
