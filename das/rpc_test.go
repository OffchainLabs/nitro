// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/offchainlabs/nitro/blsSignatures"
	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/util/signature"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func blsPubToBase64(pubkey *blsSignatures.PublicKey) string {
	pubkeyBytes := blsSignatures.PublicKeyToBytes(*pubkey)
	encodedPubkey := make([]byte, base64.StdEncoding.EncodedLen(len(pubkeyBytes)))
	base64.StdEncoding.Encode(encodedPubkey, pubkeyBytes)
	return string(encodedPubkey)
}

type sleepOnIterationFn func(i int)

func testRpcImpl(t *testing.T, size, times int, concurrent bool) {
	// enableLogging()

	ctx := context.Background()
	lis, err := net.Listen("tcp", "localhost:0")
	testhelpers.RequireImpl(t, err)
	keyDir := t.TempDir()
	dataDir := t.TempDir()
	pubkey, _, err := GenerateAndStoreKeys(keyDir)
	testhelpers.RequireImpl(t, err)

	config := DataAvailabilityConfig{
		Enable: true,
		Key: KeyConfig{
			KeyDir: keyDir,
		},
		LocalFileStorage: LocalFileStorageConfig{
			Enable:  true,
			DataDir: dataDir,
		},
		ParentChainNodeURL: "none",
		RequestTimeout:     5 * time.Second,
	}

	var syncFromStorageServices []*IterableStorageService
	var syncToStorageServices []StorageService
	storageService, lifecycleManager, err := CreatePersistentStorageService(ctx, &config, &syncFromStorageServices, &syncToStorageServices)
	testhelpers.RequireImpl(t, err)
	defer lifecycleManager.StopAndWaitUntil(time.Second)
	localDas, err := NewSignAfterStoreDASWriter(ctx, config, storageService)
	testhelpers.RequireImpl(t, err)

	testPrivateKey, err := crypto.GenerateKey()
	testhelpers.RequireImpl(t, err)

	signatureVerifier, err := NewSignatureVerifierWithSeqInboxCaller(nil, "0x"+hex.EncodeToString(crypto.FromECDSAPub(&testPrivateKey.PublicKey)))
	testhelpers.RequireImpl(t, err)
	signer := signature.DataSignerFromPrivateKey(testPrivateKey)

	dasServer, err := StartDASRPCServerOnListener(ctx, lis, genericconf.HTTPServerTimeoutConfigDefault, genericconf.HTTPServerBodyLimitDefault, storageService, localDas, storageService, signatureVerifier)

	defer func() {
		if err := dasServer.Shutdown(ctx); err != nil {
			panic(err)
		}
	}()
	testhelpers.RequireImpl(t, err)
	beConfig := BackendConfig{
		URL:                 "http://" + lis.Addr().String(),
		PubKeyBase64Encoded: blsPubToBase64(pubkey),
		SignerMask:          1,
	}

	backendsJsonByte, err := json.Marshal([]BackendConfig{beConfig})
	testhelpers.RequireImpl(t, err)
	aggConf := DataAvailabilityConfig{
		RPCAggregator: AggregatorConfig{
			AssumedHonest:         1,
			Backends:              string(backendsJsonByte),
			MaxStoreChunkBodySize: (chunkSize * 2) + len(sendChunkJSONBoilerplate),
		},
		RequestTimeout: time.Minute,
	}
	rpcAgg, err := NewRPCAggregatorWithSeqInboxCaller(aggConf, nil, signer)
	testhelpers.RequireImpl(t, err)

	var wg sync.WaitGroup
	runStore := func() {
		defer wg.Done()
		msg := testhelpers.RandomizeSlice(make([]byte, size))
		cert, err := rpcAgg.Store(ctx, msg, 0, nil)
		testhelpers.RequireImpl(t, err)

		retrievedMessage, err := storageService.GetByHash(ctx, cert.DataHash)
		testhelpers.RequireImpl(t, err)

		if !bytes.Equal(msg, retrievedMessage) {
			testhelpers.FailImpl(t, "failed to retrieve correct message")
		}

		retrievedMessage, err = storageService.GetByHash(ctx, cert.DataHash)
		testhelpers.RequireImpl(t, err)

		if !bytes.Equal(msg, retrievedMessage) {
			testhelpers.FailImpl(t, "failed to getByHash correct message")
		}
	}

	for i := 0; i < times; i++ {
		wg.Add(1)
		if concurrent {
			go runStore()
		} else {
			runStore()
		}
	}

	wg.Wait()
}

const chunkSize = 512 * 1024

func TestRPCStore(t *testing.T) {
	for _, tc := range []struct {
		desc             string
		totalSize, times int
		concurrent       bool
		leagcyAPIOnly    bool
	}{
		{desc: "small store", totalSize: 100, times: 1, concurrent: false},
		{desc: "chunked store - last chunk full", totalSize: chunkSize * 20, times: 10, concurrent: true},
		{desc: "chunked store - last chunk not full", totalSize: chunkSize*31 + 123, times: 10, concurrent: true},
		{desc: "chunked store - overflow cache - sequential", totalSize: chunkSize * 3, times: 15, concurrent: false},
		{desc: "new client falls back to old api for old server", totalSize: (5*1024*1024)/2 - len(sendChunkJSONBoilerplate) - 100 /* geth counts headers too */, times: 5, concurrent: true, leagcyAPIOnly: true},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			legacyDASStoreAPIOnly = tc.leagcyAPIOnly
			testRpcImpl(t, tc.totalSize, tc.times, tc.concurrent)
		})
	}
}
