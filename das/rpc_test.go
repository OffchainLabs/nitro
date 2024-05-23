// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net"
	"testing"
	"time"

	"github.com/offchainlabs/nitro/blsSignatures"
	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func blsPubToBase64(pubkey *blsSignatures.PublicKey) string {
	pubkeyBytes := blsSignatures.PublicKeyToBytes(*pubkey)
	encodedPubkey := make([]byte, base64.StdEncoding.EncodedLen(len(pubkeyBytes)))
	base64.StdEncoding.Encode(encodedPubkey, pubkeyBytes)
	return string(encodedPubkey)
}

func TestRPC(t *testing.T) {
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
	dasServer, err := StartDASRPCServerOnListener(ctx, lis, genericconf.HTTPServerTimeoutConfigDefault, storageService, localDas, storageService, &SignatureVerifier{})
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
			AssumedHonest: 1,
			Backends:      string(backendsJsonByte),
		},
		RequestTimeout: 5 * time.Second,
	}
	rpcAgg, err := NewRPCAggregatorWithSeqInboxCaller(aggConf, nil, nil)
	testhelpers.RequireImpl(t, err)

	msg := testhelpers.RandomizeSlice(make([]byte, 100))
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
