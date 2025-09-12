// Copyright 2024-2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package daclient

import (
	"context"
	"fmt"

	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/daprovider/das"
	"github.com/offchainlabs/nitro/util/rpcclient"
	"github.com/offchainlabs/nitro/util/signature"
)

type Client struct {
	*rpcclient.RpcClient
	*das.DataStreamer
}

type ClientConfig struct {
	Enable          bool                   `koanf:"enable"`
	WithWriter      bool                   `koanf:"with-writer"`
	RPC             rpcclient.ClientConfig `koanf:"rpc"`
	MaxPostBodySize int                    `koanf:"max-post-body-size"`
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
	MaxPostBodySize: genericconf.HTTPServerBodyLimitDefault,
}

func ClientConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".enable", DefaultClientConfig.Enable, "enable daprovider client")
	f.Bool(prefix+".with-writer", DefaultClientConfig.WithWriter, "implies if the daprovider rpc server supports writer interface")
	rpcclient.RPCClientAddOptions(prefix+".rpc", f, &DefaultClientConfig.RPC)
	f.Int(prefix+".max-post-body-size", DefaultClientConfig.MaxPostBodySize, "max HTTP POST body size")
}

func NewClient(ctx context.Context, config rpcclient.ClientConfigFetcher, maxPostBodySize int, dataSigner signature.DataSignerFunc) (*Client, error) {
	storeRpcMethods := das.DataStreamingRPCMethods{
		StartReceiving:    "daprovider_startChunkedStore",
		ReceiveChunk:      "daprovider_sendChunk",
		FinalizeReceiving: "daprovider_commitChunkedStore",
	}
	dataStreamer, err := das.NewDataStreamer(config().URL, maxPostBodySize, dataSigner, storeRpcMethods)
	if err != nil {
		return nil, err
	}

	client := &Client{rpcclient.NewRpcClient(config, nil), dataStreamer}
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
	Payload   hexutil.Bytes           `json:"payload,omitempty"`
	Preimages daprovider.PreimagesMap `json:"preimages,omitempty"`
}

func (c *Client) RecoverPayloadFromBatch(
	ctx context.Context,
	batchNum uint64,
	batchBlockHash common.Hash,
	sequencerMsg []byte,
	preimages daprovider.PreimagesMap,
	validateSeqMsg bool,
) ([]byte, daprovider.PreimagesMap, error) {
	var recoverPayloadFromBatchResult RecoverPayloadFromBatchResult
	if err := c.CallContext(ctx, &recoverPayloadFromBatchResult, "daprovider_recoverPayloadFromBatch", hexutil.Uint64(batchNum), batchBlockHash, hexutil.Bytes(sequencerMsg), preimages, validateSeqMsg); err != nil {
		return nil, nil, fmt.Errorf("error returned from daprovider_recoverPayloadFromBatch rpc method, err: %w", err)
	}
	return recoverPayloadFromBatchResult.Payload, recoverPayloadFromBatchResult.Preimages, nil
}

type StartChunkedStoreResult struct {
	MessageId hexutil.Uint64 `json:"messageId,omitempty"`
}

type SendChunkResult struct {
	Ok hexutil.Uint64 `json:"sendChunkResult,omitempty"`
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
	_, err := c.DataStreamer.StreamData(ctx, message, timeout)
	if err != nil {
		// TODO? try store on chain
		return nil, fmt.Errorf("error returned from daprovider rpc, err: %w", err)
	}
	return make([]byte, 0), nil
}

// GenerateProofResult is the result struct that data availability providers should use to respond with a proof for a specific preimage
type GenerateProofResult struct {
	Proof hexutil.Bytes `json:"proof,omitempty"`
}

// GenerateProof generates a proof for a specific preimage at a given offset
// This method calls the external DA provider's RPC endpoint to generate the proof
func (c *Client) GenerateProof(
	ctx context.Context,
	preimageType arbutil.PreimageType,
	certHash common.Hash,
	offset uint64,
	certificate []byte,
) ([]byte, error) {
	var generateProofResult GenerateProofResult
	if err := c.CallContext(ctx, &generateProofResult, "daprovider_generateProof", hexutil.Uint(preimageType), certHash, hexutil.Uint64(offset), hexutil.Bytes(certificate)); err != nil {
		return nil, fmt.Errorf("error returned from daprovider_generateProof rpc method, err: %w", err)
	}
	return generateProofResult.Proof, nil
}

// GenerateCertificateValidityProofResult is the result struct that data availability providers should use to respond with validity proof
type GenerateCertificateValidityProofResult struct {
	Proof hexutil.Bytes `json:"proof,omitempty"`
}

func (c *Client) GenerateCertificateValidityProof(
	ctx context.Context,
	preimageType arbutil.PreimageType,
	certificate []byte,
) ([]byte, error) {
	var generateCertificateValidityProofResult GenerateCertificateValidityProofResult
	if err := c.CallContext(ctx, &generateCertificateValidityProofResult, "daprovider_generateCertificateValidityProof", hexutil.Uint(preimageType), hexutil.Bytes(certificate)); err != nil {
		return nil, fmt.Errorf("error returned from daprovider_generateCertificateValidityProof rpc method, err: %w", err)
	}
	return generateCertificateValidityProofResult.Proof, nil
}
