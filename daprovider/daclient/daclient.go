// Copyright 2024-2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package daclient

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/knadh/koanf"
	"github.com/knadh/koanf/providers/confmap"
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
	rpcTimeout     time.Duration
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

type ExternalProviderConfigList []ClientConfig

func (l *ExternalProviderConfigList) String() string {
	b, _ := json.Marshal(*l)
	return string(b)
}

func (l *ExternalProviderConfigList) Set(value string) error {
	return l.UnmarshalJSON([]byte(value))
}

func (l *ExternalProviderConfigList) UnmarshalJSON(data []byte) error {
	var tmp []ClientConfig
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	*l = tmp
	return nil
}

func (l *ExternalProviderConfigList) Type() string {
	return "externalProviderConfigList"
}

var parsedExternalProvidersConf ExternalProviderConfigList

func ClientConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".enable", DefaultClientConfig.Enable, "enable daprovider client")
	f.Bool(prefix+".with-writer", DefaultClientConfig.WithWriter, "implies if the daprovider rpc server supports writer interface")
	rpcclient.RPCClientAddOptions(prefix+".rpc", f, &DefaultClientConfig.RPC)
	f.Bool(prefix+".use-data-streaming", DefaultClientConfig.UseDataStreaming, "use data streaming protocol for storing large payloads")
	data_streaming.DataStreamerConfigAddOptions(prefix+".data-stream", f, DefaultStreamRpcMethods)
	f.String(prefix+".store-rpc-method", DefaultClientConfig.StoreRpcMethod, "name of the store rpc method on the daprovider server (used when data streaming is disabled)")
}

func ExternalProviderConfigAddPluralOptions(prefix string, f *pflag.FlagSet) {
	f.Var(&parsedExternalProvidersConf, prefix+"s", "JSON array of external DA provider configurations. This can be specified on the command line as a JSON array, eg: [{\"rpc\":{\"url\":\"...\"},\"with-writer\":true},...], or as a JSON array in the config file.")
}

func FixExternalProvidersCLIParsing(path string, k *koanf.Koanf) error {
	rawProviders := k.Get(path)
	if providers, ok := rawProviders.(string); ok {
		err := parsedExternalProvidersConf.UnmarshalJSON([]byte(providers))
		if err != nil {
			return err
		}

		tempMap := map[string]interface{}{
			path: parsedExternalProvidersConf,
		}

		if err = k.Load(confmap.Provider(tempMap, "."), nil); err != nil {
			return err
		}
	}
	return nil
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
		rpcTimeout:     config.RPC.Timeout,
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
	promise, ctx := containers.NewPromiseWithContext[SupportedHeaderBytesResult](context.Background())
	go func() {
		var result server_api.SupportedHeaderBytesResult
		if err := c.CallContext(ctx, &result, "daprovider_getSupportedHeaderBytes"); err != nil {
			promise.ProduceError(fmt.Errorf("error returned from daprovider_getSupportedHeaderBytes rpc method: %w", err))
		} else {
			promise.Produce(SupportedHeaderBytesResult{HeaderBytes: result.HeaderBytes})
		}
	}()
	return promise
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
	message []byte,
	timeout uint64,
) containers.PromiseInterface[[]byte] {
	var promise *containers.Promise[[]byte]
	var ctx context.Context

	// Create context with timeout if configured, otherwise use background
	if c.rpcTimeout > 0 {
		timeoutCtx, cancel := context.WithTimeout(context.Background(), c.rpcTimeout)
		p := containers.NewPromise[[]byte](cancel)
		promise = &p
		ctx = timeoutCtx
	} else {
		promise, ctx = containers.NewPromiseWithContext[[]byte](context.Background())
	}

	go func() {
		storeResult, err := c.store(ctx, message, timeout)
		if err != nil {
			promise.ProduceError(err)
		} else {
			promise.Produce(storeResult.SerializedDACert)
		}
	}()
	return promise
}

func (c *Client) store(ctx context.Context, message []byte, timeout uint64) (*server_api.StoreResult, error) {
	var storeResult *server_api.StoreResult

	// Single-call store if data streaming is not enabled
	if c.DataStreamer == nil {
		if err := c.CallContext(ctx, &storeResult, *c.storeRpcMethod, hexutil.Bytes(message), hexutil.Uint64(timeout)); err != nil {
			return nil, fmt.Errorf("error returned from daprovider server (single-call store protocol), err: %w", err)
		}
		return storeResult, nil
	}

	// Otherwise, use the data streaming protocol
	storeResult, err := c.DataStreamer.StreamData(ctx, message, timeout)
	if err != nil {
		return nil, fmt.Errorf("error returned from daprovider server (chunked store protocol), err: %w", err)
	} else {
		return storeResult, nil
	}
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
