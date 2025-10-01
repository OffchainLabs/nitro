// Copyright 2024-2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package daclient

import (
	"context"
	"fmt"

	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/daprovider/server_api"
	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/rpcclient"
)

type Client struct {
	*rpcclient.RpcClient
}

type ClientConfig struct {
	Enable     bool                   `koanf:"enable"`
	WithWriter bool                   `koanf:"with-writer"`
	RPC        rpcclient.ClientConfig `koanf:"rpc" reload:"hot"`
}

var DefaultClientConfig = ClientConfig{
	Enable:     false,
	WithWriter: false,
	RPC: rpcclient.ClientConfig{
		Retries:                   3,
		RetryErrors:               "websocket: close.*|dial tcp .*|.*i/o timeout|.*connection reset by peer|.*connection refused",
		ArgLogLimit:               2048,
		WebsocketMessageSizeLimit: 256 * 1024 * 1024,
	},
}

func ClientConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".enable", DefaultClientConfig.Enable, "enable daprovider client")
	f.Bool(prefix+".with-writer", DefaultClientConfig.WithWriter, "implies if the daprovider rpc server supports writer interface")
	rpcclient.RPCClientAddOptions(prefix+".rpc", f, &DefaultClientConfig.RPC)
}

func NewClient(ctx context.Context, config rpcclient.ClientConfigFetcher) (*Client, error) {
	client := &Client{rpcclient.NewRpcClient(config, nil)}
	if err := client.Start(ctx); err != nil {
		return nil, fmt.Errorf("error starting daprovider client: %w", err)
	}
	return client, nil
}

func (c *Client) GetSupportedHeaderBytes(ctx context.Context) ([]byte, error) {
	var result server_api.SupportedHeaderBytesResult
	if err := c.CallContext(ctx, &result, "daprovider_getSupportedHeaderBytes"); err != nil {
		return nil, fmt.Errorf("error returned from daprovider_getSupportedHeaderBytes rpc method: %w", err)
	}
	return result.HeaderBytes, nil
}

// RecoverPayload fetches the underlying payload from the DA provider
func (c *Client) RecoverPayload(
	batchNum uint64,
	batchBlockHash common.Hash,
	sequencerMsg []byte,
) containers.PromiseInterface[daprovider.PayloadResult] {
	promise, ctx := containers.NewPromiseWithContext[daprovider.PayloadResult](context.Background())
	go func() {
		var result daprovider.PayloadResult
		if err := c.CallContext(ctx, &result, "daprovider_recoverPayload", hexutil.Uint64(batchNum), batchBlockHash, hexutil.Bytes(sequencerMsg)); err != nil {
			promise.ProduceError(fmt.Errorf("error returned from daprovider_recoverPayload rpc method, err: %w", err))
		} else {
			promise.Produce(result)
		}
	}()
	return promise
}

// CollectPreimages collects preimages from the DA provider
func (c *Client) CollectPreimages(
	batchNum uint64,
	batchBlockHash common.Hash,
	sequencerMsg []byte,
) containers.PromiseInterface[daprovider.PreimagesResult] {
	promise, ctx := containers.NewPromiseWithContext[daprovider.PreimagesResult](context.Background())
	go func() {
		var result daprovider.PreimagesResult
		if err := c.CallContext(ctx, &result, "daprovider_collectPreimages", hexutil.Uint64(batchNum), batchBlockHash, hexutil.Bytes(sequencerMsg)); err != nil {
			promise.ProduceError(fmt.Errorf("error returned from daprovider_collectPreimages rpc method, err: %w", err))
		} else {
			promise.Produce(result)
		}
	}()
	return promise
}

func (c *Client) Store(
	ctx context.Context,
	message []byte,
	timeout uint64,
	disableFallbackStoreDataOnChain bool,
) ([]byte, error) {
	var storeResult server_api.StoreResult
	if err := c.CallContext(ctx, &storeResult, "daprovider_store", hexutil.Bytes(message), hexutil.Uint64(timeout), disableFallbackStoreDataOnChain); err != nil {
		return nil, fmt.Errorf("error returned from daprovider_store rpc method, err: %w", err)
	}
	return storeResult.SerializedDACert, nil
}

// GenerateReadPreimageProof generates a proof for a specific preimage at a given offset
// This method calls the external DA provider's RPC endpoint to generate the proof
func (c *Client) GenerateReadPreimageProof(
	certHash common.Hash,
	offset uint64,
	certificate []byte,
) containers.PromiseInterface[daprovider.PreimageProofResult] {
	promise, ctx := containers.NewPromiseWithContext[daprovider.PreimageProofResult](context.Background())
	go func() {
		var generateProofResult server_api.GenerateReadPreimageProofResult
		if err := c.CallContext(ctx, &generateProofResult, "daprovider_generateReadPreimageProof", certHash, hexutil.Uint64(offset), hexutil.Bytes(certificate)); err != nil {
			promise.ProduceError(fmt.Errorf("error returned from daprovider_generateProof rpc method, err: %w", err))
		} else {
			promise.Produce(daprovider.PreimageProofResult{Proof: generateProofResult.Proof})
		}
	}()
	return promise
}

func (c *Client) GenerateCertificateValidityProof(
	certificate []byte,
) containers.PromiseInterface[daprovider.ValidityProofResult] {
	promise, ctx := containers.NewPromiseWithContext[daprovider.ValidityProofResult](context.Background())
	go func() {
		var generateCertificateValidityProofResult server_api.GenerateCertificateValidityProofResult
		if err := c.CallContext(ctx, &generateCertificateValidityProofResult, "daprovider_generateCertificateValidityProof", hexutil.Bytes(certificate)); err != nil {
			promise.ProduceError(fmt.Errorf("error returned from daprovider_generateCertificateValidityProof rpc method, err: %w", err))
		} else {
			promise.Produce(daprovider.ValidityProofResult{Proof: generateCertificateValidityProofResult.Proof})
		}
	}()
	return promise
}
