package dasrpc

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net"
	"testing"

	"github.com/offchainlabs/nitro/blsSignatures"
	"github.com/offchainlabs/nitro/das"
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
	pubkey, _, err := das.GenerateAndStoreKeys(keyDir)
	testhelpers.RequireImpl(t, err)
	dasConfig := das.LocalDiskDASConfig{
		KeyDir:  keyDir,
		DataDir: dataDir,
	}
	localDas, err := das.NewLocalDiskDASWithSeqInboxCaller(dasConfig, nil)
	testhelpers.RequireImpl(t, err)
	dasServer, err := StartDASRPCServerOnListener(ctx, lis, localDas)
	defer func() {
		if err := dasServer.Shutdown(ctx); err != nil {
			panic(err)
		}
	}()
	testhelpers.RequireImpl(t, err)
	config := BackendConfig{
		URL:                 "http://" + lis.Addr().String(),
		PubKeyBase64Encoded: blsPubToBase64(pubkey),
		SignerMask:          1,
	}

	backendsJsonByte, err := json.Marshal([]BackendConfig{config})
	testhelpers.RequireImpl(t, err)
	aggConf := das.AggregatorConfig{
		AssumedHonest: 1,
		Backends:      string(backendsJsonByte),
	}
	rpcAgg, err := NewRPCAggregatorWithSeqInboxCaller(aggConf, nil)
	testhelpers.RequireImpl(t, err)

	msg := testhelpers.RandomizeSlice(make([]byte, 100))
	cert, err := rpcAgg.Store(ctx, msg, 0, nil)
	testhelpers.RequireImpl(t, err)

	retrievedMessage, err := rpcAgg.Retrieve(ctx, cert)
	testhelpers.RequireImpl(t, err)

	if !bytes.Equal(msg, retrievedMessage) {
		testhelpers.FailImpl(t, "failed to retrieve correct message")
	}

	retrievedMessage, err = rpcAgg.GetByHash(ctx, cert.DataHash[:])
	testhelpers.RequireImpl(t, err)

	if !bytes.Equal(msg, retrievedMessage) {
		testhelpers.FailImpl(t, "failed to getByHash correct message")
	}
}
