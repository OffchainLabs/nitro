package daclient

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util/rpcclient"
	"github.com/spf13/pflag"
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

// IsValidHeaderByteResult is the result struct that data availability providers should use to respond if the given headerByte corresponds to their DA service
type IsValidHeaderByteResult struct {
	IsValid bool `json:"is-valid,omitempty"`
}

func (c *Client) IsValidHeaderByte(ctx context.Context, headerByte byte) bool {
	var isValidHeaderByteResult IsValidHeaderByteResult
	if err := c.CallContext(ctx, &isValidHeaderByteResult, "daprovider_isValidHeaderByte", headerByte); err != nil {
		log.Error("Error returned from daprovider_isValidHeaderByte rpc method, defaulting to result as false", "err", err)
		return false
	}
	return isValidHeaderByteResult.IsValid
}

// RecoverPayloadFromBatchResult is the result struct that data availability providers should use to respond with underlying payload and updated preimages map to a RecoverPayloadFromBatch fetch request
type RecoverPayloadFromBatchResult struct {
	Payload   hexutil.Bytes                                   `json:"payload,omitempty"`
	Preimages map[arbutil.PreimageType]map[common.Hash][]byte `json:"preimages,omitempty"`
}

func (c *Client) RecoverPayloadFromBatch(
	ctx context.Context,
	batchNum uint64,
	batchBlockHash common.Hash,
	sequencerMsg []byte,
	preimages map[arbutil.PreimageType]map[common.Hash][]byte,
	validateSeqMsg bool,
) ([]byte, map[arbutil.PreimageType]map[common.Hash][]byte, error) {
	var recoverPayloadFromBatchResult RecoverPayloadFromBatchResult
	if err := c.CallContext(ctx, &recoverPayloadFromBatchResult, "daprovider_recoverPayloadFromBatch", hexutil.Uint64(batchNum), batchBlockHash, hexutil.Bytes(sequencerMsg), preimages, validateSeqMsg); err != nil {
		return nil, nil, fmt.Errorf("error returned from daprovider_recoverPayloadFromBatch rpc method, err: %w", err)
	}
	return recoverPayloadFromBatchResult.Payload, recoverPayloadFromBatchResult.Preimages, nil
}

// StoreResult is the result struct that data availability providers should use to respond with a commitment to a Store request for posting batch data to their DA service
type StoreResult struct {
	SerializedDACert hexutil.Bytes `json:"serialized-da-cert,omitempty"`
}

func (c *Client) Store(
	ctx context.Context,
	message []byte,
	timeout uint64,
	disableFallbackStoreDataOnChain bool,
) ([]byte, error) {
	var storeResult StoreResult
	if err := c.CallContext(ctx, &storeResult, "daprovider_store", hexutil.Bytes(message), hexutil.Uint64(timeout), disableFallbackStoreDataOnChain); err != nil {
		return nil, fmt.Errorf("error returned from daprovider_store rpc method, err: %w", err)
	}
	return storeResult.SerializedDACert, nil
}
