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
	"github.com/offchainlabs/nitro/daprovider/das/dasutil"
	testflag "github.com/offchainlabs/nitro/util/testhelpers/flag"
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

		config := DataAvailabilityConfig{
			Enable: true,
			Key: KeyConfig{
				PrivKey: privKey,
			},
			ParentChainNodeURL: "none",
		}

		storageServices = append(storageServices, NewMemoryBackedStorageService(ctx))
		das, err := NewSignAfterStoreDASWriter(ctx, config, storageServices[i])
		Require(t, err)
		signerMask := uint64(1 << i)
		details, err := NewServiceDetails(das, *das.pubKey, signerMask, "service"+strconv.Itoa(i))
		Require(t, err)
		backends = append(backends, *details)
	}

	aggregator, err := NewAggregator(ctx, DataAvailabilityConfig{RPCAggregator: AggregatorConfig{AssumedHonest: 1, EnableChunkedStore: true}, ParentChainNodeURL: "none"}, backends)
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
	DataAvailabilityServiceWriter
}

func (w *WrapStore) Store(ctx context.Context, message []byte, timeout uint64) (*dasutil.DataAvailabilityCertificate, error) {
	switch w.injector.shouldFail() {
	case success:
		return w.DataAvailabilityServiceWriter.Store(ctx, message, timeout)
	case immediateError:
		return nil, errors.New("expected Store failure")
	case tooSlow:
		<-ctx.Done()
		return nil, ctx.Err()
	case dataCorruption:
		cert, err := w.DataAvailabilityServiceWriter.Store(ctx, message, timeout)
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

		config := DataAvailabilityConfig{
			Enable: true,
			Key: KeyConfig{
				PrivKey: privKey,
			},
			ParentChainNodeURL: "none",
		}

		storageServices = append(storageServices, NewMemoryBackedStorageService(ctx))
		das, err := NewSignAfterStoreDASWriter(ctx, config, storageServices[i])
		Require(t, err)
		signerMask := uint64(1 << i)
		details, err := NewServiceDetails(&WrapStore{t, injectedFailures, das}, *das.pubKey, signerMask, "service"+strconv.Itoa(i))
		Require(t, err)
		backends = append(backends, *details)
	}

	aggregator, err := NewAggregator(
		ctx,
		DataAvailabilityConfig{
			RPCAggregator:      AggregatorConfig{AssumedHonest: assumedHonest, EnableChunkedStore: true},
			ParentChainNodeURL: "none",
			RequestTimeout:     time.Millisecond * 2000,
		}, backends)
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
