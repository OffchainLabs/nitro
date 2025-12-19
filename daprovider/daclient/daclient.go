// Copyright 2024-2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package daclient

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/daprovider/data_streaming"
	"github.com/offchainlabs/nitro/daprovider/server_api"
	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/rpcclient"
)

// lint:require-exhaustive-initialization
type Client struct {
	*rpcclient.RpcClient
	*data_streaming.DataStreamer[server_api.StoreResult]
	storeRpcMethod *string
}

// lint:require-exhaustive-initialization
type ClientConfig struct {
	Enable           bool                              `koanf:"enable"`
	WithWriter       bool                              `koanf:"with-writer"`
	RPC              rpcclient.ClientConfig            `koanf:"rpc" reload:"hot"`
	UseDataStreaming bool                              `koanf:"use-data-streaming" reload:"hot"`
	DataStream       data_streaming.DataStreamerConfig `koanf:"data-stream"`
	StoreRpcMethod   string                            `koanf:"store-rpc-method" reload:"hot"`
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
	UseDataStreaming: false,
	DataStream:       data_streaming.DefaultDataStreamerConfig(DefaultStreamRpcMethods),
	StoreRpcMethod:   DefaultStoreRpcMethod,
}

func TestClientConfig(serverUrl string) *ClientConfig {
	return &ClientConfig{
		Enable:           true,
		WithWriter:       true,
		RPC:              rpcclient.ClientConfig{URL: serverUrl},
		UseDataStreaming: false,
		DataStream:       data_streaming.TestDataStreamerConfig(DefaultStreamRpcMethods),
		StoreRpcMethod:   DefaultStoreRpcMethod,
	}
}

var DefaultStreamRpcMethods = data_streaming.DataStreamingRPCMethods{
	StartStream:    "daprovider_startChunkedStore",
	StreamChunk:    "daprovider_sendChunk",
	FinalizeStream: "daprovider_commitChunkedStore",
}

var DefaultStoreRpcMethod = "daprovider_store"

func ClientConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".enable", DefaultClientConfig.Enable, "enable daprovider client")
	f.Bool(prefix+".with-writer", DefaultClientConfig.WithWriter, "implies if the daprovider rpc server supports writer interface")
	rpcclient.RPCClientAddOptions(prefix+".rpc", f, &DefaultClientConfig.RPC)
	f.Bool(prefix+".use-data-streaming", DefaultClientConfig.UseDataStreaming, "use data streaming protocol for storing large payloads")
	data_streaming.DataStreamerConfigAddOptions(prefix+".data-stream", f, DefaultStreamRpcMethods)
	f.String(prefix+".store-rpc-method", DefaultClientConfig.StoreRpcMethod, "name of the store rpc method on the daprovider server (used when data streaming is disabled)")
}

func NewClient(ctx context.Context, config *ClientConfig, payloadSigner *data_streaming.PayloadSigner) (*Client, error) {
	rpcClient := rpcclient.NewRpcClient(func() *rpcclient.ClientConfig { return &config.RPC }, nil)
	var err error

	var dataStreamer *data_streaming.DataStreamer[server_api.StoreResult]
	if config.UseDataStreaming {
		dataStreamer, err = data_streaming.NewDataStreamer[server_api.StoreResult](config.DataStream, payloadSigner, rpcClient)
		if err != nil {
			return nil, err
		}
	}

	client := &Client{
		RpcClient:      rpcClient,
		DataStreamer:   dataStreamer,
		storeRpcMethod: &config.StoreRpcMethod,
	}
	if err = client.Start(ctx); err != nil {
		return nil, fmt.Errorf("error starting daprovider client: %w", err)
	}
	return client, nil
}

type SupportedHeaderBytesResult struct {
	HeaderBytes []byte
}

func (c *Client) GetSupportedHeaderBytes() containers.PromiseInterface[SupportedHeaderBytesResult] {
	return containers.DoPromise(context.Background(), func(ctx context.Context) (SupportedHeaderBytesResult, error) {
		var result server_api.SupportedHeaderBytesResult
		if err := c.CallContext(ctx, &result, "daprovider_getSupportedHeaderBytes"); err != nil {
			return SupportedHeaderBytesResult{}, fmt.Errorf("error returned from daprovider_getSupportedHeaderBytes rpc method: %w", err)
		}
		return SupportedHeaderBytesResult{HeaderBytes: result.HeaderBytes}, nil
	})
}

func (c *Client) GetMaxMessageSize() containers.PromiseInterface[int] {
	return containers.DoPromise(context.Background(), func(ctx context.Context) (int, error) {
		var result server_api.MaxMessageSizeResult
		if err := c.CallContext(ctx, &result, "daprovider_getMaxMessageSize"); err != nil {
			return 0, fmt.Errorf("error returned from daprovider_getMaxMessageSize rpc method: %w", err)
		}
		return result.MaxSize, nil
	})
}

// RecoverPayload fetches the underlying payload from the DA provider
func (c *Client) RecoverPayload(
	batchNum uint64,
	batchBlockHash common.Hash,
	sequencerMsg []byte,
) containers.PromiseInterface[daprovider.PayloadResult] {
	return containers.DoPromise(context.Background(), func(ctx context.Context) (daprovider.PayloadResult, error) {
		var result daprovider.PayloadResult
		err := c.CallContext(ctx, &result, "daprovider_recoverPayload", hexutil.Uint64(batchNum), batchBlockHash, hexutil.Bytes(sequencerMsg))
		if err != nil {
			err = fmt.Errorf("error returned from daprovider_recoverPayload rpc method, err: %w", err)
		}
		return result, err
	})
}

// CollectPreimages collects preimages from the DA provider
func (c *Client) CollectPreimages(
	batchNum uint64,
	batchBlockHash common.Hash,
	sequencerMsg []byte,
) containers.PromiseInterface[daprovider.PreimagesResult] {
	return containers.DoPromise(context.Background(), func(ctx context.Context) (daprovider.PreimagesResult, error) {
		var result daprovider.PreimagesResult
		err := c.CallContext(ctx, &result, "daprovider_collectPreimages", hexutil.Uint64(batchNum), batchBlockHash, hexutil.Bytes(sequencerMsg))
		if err != nil {
			err = fmt.Errorf("error returned from daprovider_collectPreimages rpc method, err: %w", err)
		}
		return result, err
	})
}

func (c *Client) Store(
	message []byte,
	timeout uint64,
) containers.PromiseInterface[[]byte] {
	return containers.DoPromise(context.Background(), func(ctx context.Context) ([]byte, error) {
		storeResult, err := c.store(ctx, message, timeout)
		if err != nil {
			return nil, err
		}
		return storeResult.SerializedDACert, nil
	})
}

func (c *Client) store(ctx context.Context, message []byte, timeout uint64) (*server_api.StoreResult, error) {
	var storeResult *server_api.StoreResult

	// Single-call store if data streaming is not enabled
	if c.DataStreamer == nil {
		if err := c.CallContext(ctx, &storeResult, *c.storeRpcMethod, hexutil.Bytes(message), hexutil.Uint64(timeout)); err != nil {
			// Restore error identity lost over RPC for external DA providers
			// by wrapping the original error so we preserve context for debugging
			if strings.Contains(err.Error(), daprovider.ErrFallbackRequested.Error()) {
				return nil, fmt.Errorf("%w (from external DA provider: %w)", daprovider.ErrFallbackRequested, err)
			}
			if strings.Contains(err.Error(), daprovider.ErrMessageTooLarge.Error()) {
				return nil, fmt.Errorf("%w (from external DA provider: %w)", daprovider.ErrMessageTooLarge, err)
			}
			return nil, fmt.Errorf("error returned from daprovider server (single-call store protocol), err: %w", err)
		}
		return storeResult, nil
	}

	// Otherwise, use the data streaming protocol
	storeResult, err := c.DataStreamer.StreamData(ctx, message, timeout)
	if err != nil {
		// Restore error identity lost over RPC for external DA providers
		// by wrapping the original error so we preserve context for debugging
		if strings.Contains(err.Error(), daprovider.ErrFallbackRequested.Error()) {
			return nil, fmt.Errorf("%w (from external DA provider: %w)", daprovider.ErrFallbackRequested, err)
		}
		if strings.Contains(err.Error(), daprovider.ErrMessageTooLarge.Error()) {
			return nil, fmt.Errorf("%w (from external DA provider: %w)", daprovider.ErrMessageTooLarge, err)
		}
		return nil, fmt.Errorf("error returned from daprovider server (chunked store protocol), err: %w", err)
	}
	return storeResult, nil
}

// GenerateReadPreimageProof generates a proof for a specific preimage at a given offset
// This method calls the external DA provider's RPC endpoint to generate the proof
func (c *Client) GenerateReadPreimageProof(
	offset uint64,
	certificate []byte,
) containers.PromiseInterface[daprovider.PreimageProofResult] {
	return containers.DoPromise(context.Background(), func(ctx context.Context) (daprovider.PreimageProofResult, error) {
		var generateProofResult server_api.GenerateReadPreimageProofResult
		if err := c.CallContext(ctx, &generateProofResult, "daprovider_generateReadPreimageProof", hexutil.Uint64(offset), hexutil.Bytes(certificate)); err != nil {
			return daprovider.PreimageProofResult{}, fmt.Errorf("error returned from daprovider_generateProof rpc method, err: %w", err)
		}
		return daprovider.PreimageProofResult{Proof: generateProofResult.Proof}, nil
	})
}

func (c *Client) GenerateCertificateValidityProof(
	certificate []byte,
) containers.PromiseInterface[daprovider.ValidityProofResult] {
	return containers.DoPromise(context.Background(), func(ctx context.Context) (daprovider.ValidityProofResult, error) {
		var generateCertificateValidityProofResult server_api.GenerateCertificateValidityProofResult
		if err := c.CallContext(ctx, &generateCertificateValidityProofResult, "daprovider_generateCertificateValidityProof", hexutil.Bytes(certificate)); err != nil {
			return daprovider.ValidityProofResult{}, fmt.Errorf("error returned from daprovider_generateCertificateValidityProof rpc method, err: %w", err)
		}
		return daprovider.ValidityProofResult{Proof: generateCertificateValidityProofResult.Proof}, nil
	})
}
