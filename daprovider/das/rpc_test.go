// Copyright 2021-2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package das

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/offchainlabs/nitro/blsSignatures"
	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/daprovider/data_streaming"
	"github.com/offchainlabs/nitro/util/rpcclient"
	"github.com/offchainlabs/nitro/util/signature"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func blsPubToBase64(pubkey *blsSignatures.PublicKey) string {
	pubkeyBytes := blsSignatures.PublicKeyToBytes(*pubkey)
	encodedPubkey := make([]byte, base64.StdEncoding.EncodedLen(len(pubkeyBytes)))
	base64.StdEncoding.Encode(encodedPubkey, pubkeyBytes)
	return string(encodedPubkey)
}

const sendChunkJSONBoilerplate = "{\"jsonrpc\":\"2.0\",\"id\":4294967295,\"method\":\"das_sendChunked\",\"params\":[\"\"]}"

func testRpcImpl(t *testing.T, size, times int, concurrent bool) {
	ctx := context.Background()
	lis, err := net.Listen("tcp", "localhost:0")
	testhelpers.RequireImpl(t, err)
	keyDir := t.TempDir()
	dataDir := t.TempDir()
	pubkey, _, err := GenerateAndStoreKeys(keyDir)
	testhelpers.RequireImpl(t, err)

	config := DefaultDataAvailabilityConfig
	config.Enable = true
	config.Key.KeyDir = keyDir
	config.LocalFileStorage.Enable = true
	config.LocalFileStorage.DataDir = dataDir

	storageService, lifecycleManager, err := CreatePersistentStorageService(ctx, &config)
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
	beConfigs := BackendConfigList{BackendConfig{
		URL:    "http://" + lis.Addr().String(),
		Pubkey: blsPubToBase64(pubkey),
	}}

	testhelpers.RequireImpl(t, err)
	aggConf := DefaultDataAvailabilityConfig
	aggConf.RPCAggregator.AssumedHonest = 1
	aggConf.RPCAggregator.Backends = beConfigs
	aggConf.RPCAggregator.DASRPCClient.EnableChunkedStore = true
	aggConf.RPCAggregator.DASRPCClient.DataStream = data_streaming.TestDataStreamerConfig(DefaultDataStreamRpcMethods)
	aggConf.RPCAggregator.DASRPCClient.RPC = rpcclient.TestClientConfig
	aggConf.RequestTimeout = time.Minute
	rpcAgg, err := NewRPCAggregator(aggConf, signer)
	testhelpers.RequireImpl(t, err)

	var wg sync.WaitGroup
	runStore := func() {
		defer wg.Done()
		msg := testhelpers.RandomizeSlice(make([]byte, size))
		cert, err := rpcAgg.Store(ctx, msg, testhelpers.RandomUint64(0, uint64(defaultStorageRetention.Seconds()))) // we use random timeouts as a random nonce to differentiate between request signatures to avoid replay-protection issues
		testhelpers.RequireImpl(t, err)

		retrievedMessage, err := storageService.GetByHash(ctx, cert.DataHash)
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

const chunkSize = (data_streaming.TestHttpBodyLimit - len(sendChunkJSONBoilerplate)) / 2

func TestRPCStore(t *testing.T) {
	for _, tc := range []struct {
		desc             string
		totalSize, times int
		concurrent       bool
		legacyAPIOnly    bool
	}{
		{desc: "small store", totalSize: 100, times: 1, concurrent: false},
		{desc: "chunked store - last chunk full", totalSize: chunkSize * 20, times: 10, concurrent: true},
		{desc: "chunked store - last chunk not full", totalSize: chunkSize*31 + 123, times: 10, concurrent: true},
		{desc: "chunked store - overflow cache - sequential", totalSize: chunkSize * 3, times: 15, concurrent: false},
		{desc: "new client falls back to old api for old server", totalSize: (5*1024*1024)/2 - len(sendChunkJSONBoilerplate) - 100 /* geth counts headers too */, times: 5, concurrent: true, legacyAPIOnly: true},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			legacyDASStoreAPIOnly = tc.legacyAPIOnly
			testRpcImpl(t, tc.totalSize, tc.times, tc.concurrent)
		})
	}
}
