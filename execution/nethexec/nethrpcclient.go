package nethexec

import (
	"context"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution"
)

var defaultUrl = "http://localhost:20545"

type nethRpcClient struct {
	client *rpc.Client
	url    string
}

type messageParams struct {
	Index              arbutil.MessageIndex            `json:"index"`
	Message            *arbostypes.MessageWithMetadata `json:"message"`
	MessageForPrefetch *arbostypes.MessageWithMetadata `json:"messageForPrefetch,omitempty"`
}

type initializeMessageParams struct {
	InitialL1BaseFee      *big.Int `json:"initialL1BaseFee"`
	SerializedChainConfig []byte   `json:"serializedChainConfig"`
}

type setFinalityDataParams struct {
	SafeFinalityData      *rpcFinalityData `json:"safeFinalityData,omitempty"`
	FinalizedFinalityData *rpcFinalityData `json:"finalizedFinalityData,omitempty"`
	ValidatedFinalityData *rpcFinalityData `json:"validatedFinalityData,omitempty"`
}

type rpcFinalityData struct {
	MsgIdx    uint64      `json:"msgIdx"`
	BlockHash common.Hash `json:"blockHash"`
}

type reorgParams struct {
	Index              arbutil.MessageIndex                         `json:"index"`
	Message            []arbostypes.MessageWithMetadataAndBlockInfo `json:"message"`
	MessageForPrefetch []*arbostypes.MessageWithMetadata            `json:"messageForPrefetch"`
}

type seqDelayedParams struct {
	DelayedSeqNum uint64                        `json:"delayedSeqNum"`
	Message       *arbostypes.L1IncomingMessage `json:"message"`
}

type InitMessageDigester interface {
	DigestInitMessage(ctx context.Context, initialL1BaseFee *big.Int, serializedChainConfig []byte) *execution.MessageResult
}

type fakeRemoteExecutionRpcClient struct{}

func NewFakeRemoteExecutionRpcClient() *fakeRemoteExecutionRpcClient {
	return &fakeRemoteExecutionRpcClient{}
}

func (n *fakeRemoteExecutionRpcClient) DigestInitMessage(ctx context.Context, initialL1BaseFee *big.Int, serializedChainConfig []byte) *execution.MessageResult {
	return &execution.MessageResult{}
}

var (
	_ InitMessageDigester = (*fakeRemoteExecutionRpcClient)(nil)
	_ InitMessageDigester = (*nethRpcClient)(nil)
)

func NewNethRpcClient() (*nethRpcClient, error) {
	url, exists := os.LookupEnv("PR_NETH_RPC_CLIENT_URL")
	if !exists {
		log.Warn("Wasn't able to read PR_NETH_RPC_CLIENT_URL, using default url", "url", defaultUrl)
		url = defaultUrl
	}

	httpClient := rpc.WithHTTPClient(&http.Client{
		Timeout: 30 * time.Second,
	})

	ctx := context.Background()
	rpcClient, err := rpc.DialOptions(ctx, url, httpClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create Neth RPC client: %w", err)
	}

	return &nethRpcClient{
		client: rpcClient,
		url:    url,
	}, nil
}

func (c *nethRpcClient) Close() {
	c.client.Close()
}

func (c *nethRpcClient) DigestMessage(ctx context.Context, index arbutil.MessageIndex, msg *arbostypes.MessageWithMetadata, msgForPrefetch *arbostypes.MessageWithMetadata) *execution.MessageResult {
	params := messageParams{
		Index:              index,
		Message:            msg,
		MessageForPrefetch: msgForPrefetch,
	}

	log.Info("Making JSON-RPC call to DigestMessage",
		"url", c.url,
		"index", index,
		"messageType", msg.Message.Header.Kind,
	)

	var result execution.MessageResult
	if err := c.client.CallContext(ctx, &result, "DigestMessage", params); err != nil {
		log.Error("Failed to call DigestMessage", "error", err)
		return nil
	}

	return &result
}

func (c *nethRpcClient) DigestInitMessage(ctx context.Context, initialL1BaseFee *big.Int, serializedChainConfig []byte) *execution.MessageResult {
	var result execution.MessageResult

	params := initializeMessageParams{
		InitialL1BaseFee:      initialL1BaseFee,
		SerializedChainConfig: serializedChainConfig,
	}

	log.Info("Making JSON-RPC call to DigestInitMessage",
		"url", c.url,
		"initialL1BaseFee", initialL1BaseFee,
		"len(serializedChainConfig)", len(serializedChainConfig))

	if err := c.client.CallContext(ctx, &result, "DigestInitMessage", params); err != nil {
		panic(fmt.Sprintf("failed to call DigestInitMessage: %v", err))
	}

	return &result
}

func (c *nethRpcClient) SetFinalityData(ctx context.Context, safeFinalityData *arbutil.FinalityData, finalizedFinalityData *arbutil.FinalityData, validatedFinalityData *arbutil.FinalityData) error {
	params := setFinalityDataParams{
		SafeFinalityData:      convertToRpcFinalityData(safeFinalityData),
		FinalizedFinalityData: convertToRpcFinalityData(finalizedFinalityData),
		ValidatedFinalityData: convertToRpcFinalityData(validatedFinalityData),
	}

	log.Info("Making JSON-RPC call to SetFinalityData",
		"url", c.url,
		"safeFinalityData", safeFinalityData,
		"finalizedFinalityData", finalizedFinalityData,
		"validatedFinalityData", validatedFinalityData)

	var result interface{}
	if err := c.client.CallContext(ctx, &result, "SetFinalityData", params); err != nil {
		log.Error("Failed to call SetFinalityData", "error", err)
		return fmt.Errorf("failed to call SetFinalityData: %w", err)
	}

	return nil
}

func convertToRpcFinalityData(data *arbutil.FinalityData) *rpcFinalityData {
	if data == nil {
		return nil
	}
	return &rpcFinalityData{
		MsgIdx:    uint64(data.MsgIdx),
		BlockHash: data.BlockHash,
	}
}

func (c *nethRpcClient) HeadMessageIndex(ctx context.Context) (arbutil.MessageIndex, error) {
	log.Info("Making JSON-RPC call to HeadMessageIndex", "url", c.url)
	var result hexutil.Uint64
	if err := c.client.CallContext(ctx, &result, "HeadMessageIndex"); err != nil {
		log.Error("Failed to call HeadMessageIndex", "error", err)
		return 0, fmt.Errorf("failed to call HeadMessageIndex: %w", err)
	}
	return arbutil.MessageIndex(uint64(result)), nil
}

func (c *nethRpcClient) ResultAtMessageIndex(ctx context.Context, index arbutil.MessageIndex) (*execution.MessageResult, error) {
	log.Info("Making JSON-RPC call to ResultAtMessageIndex", "url", c.url, "index", index)
	var result execution.MessageResult
	if err := c.client.CallContext(ctx, &result, "ResultAtMessageIndex", uint64(index)); err != nil {
		log.Error("Failed to call ResultAtMessageIndex", "error", err)
		return nil, fmt.Errorf("failed to call ResultAtMessageIndex: %w", err)
	}
	return &result, nil
}

func (c *nethRpcClient) MessageIndexToBlockNumber(ctx context.Context, messageIndex arbutil.MessageIndex) (uint64, error) {
	log.Info("Making JSON-RPC call to MessageIndexToBlockNumber", "url", c.url, "messageIndex", messageIndex)
	var result hexutil.Uint64
	if err := c.client.CallContext(ctx, &result, "MessageIndexToBlockNumber", uint64(messageIndex)); err != nil {
		log.Error("Failed to call MessageIndexToBlockNumber", "error", err)
		return 0, fmt.Errorf("failed to call MessageIndexToBlockNumber: %w", err)
	}
	return uint64(result), nil
}

func (c *nethRpcClient) BlockNumberToMessageIndex(ctx context.Context, blockNum uint64) (arbutil.MessageIndex, error) {
	log.Info("Making JSON-RPC call to BlockNumberToMessageIndex", "url", c.url, "blockNum", blockNum)
	var result hexutil.Uint64
	if err := c.client.CallContext(ctx, &result, "BlockNumberToMessageIndex", blockNum); err != nil {
		log.Error("Failed to call BlockNumberToMessageIndex", "error", err)
		return 0, fmt.Errorf("failed to call BlockNumberToMessageIndex: %w", err)
	}
	return arbutil.MessageIndex(uint64(result)), nil
}

func (c *nethRpcClient) MarkFeedStart(ctx context.Context, to arbutil.MessageIndex) error {
	log.Info("Making JSON-RPC call to MarkFeedStart", "url", c.url, "to", to)
	var result string
	if err := c.client.CallContext(ctx, &result, "MarkFeedStart", uint64(to)); err != nil {
		log.Error("Failed to call MarkFeedStart", "error", err)
		return fmt.Errorf("failed to call MarkFeedStart: %w", err)
	}
	return nil
}

func (c *nethRpcClient) Reorg(ctx context.Context, count arbutil.MessageIndex, newMessages []arbostypes.MessageWithMetadataAndBlockInfo, oldMessages []*arbostypes.MessageWithMetadata) ([]*execution.MessageResult, error) {
	log.Info("Making JSON-RPC call to Reorg", "url", c.url, "count", count, "newCount", len(newMessages), "oldCount", len(oldMessages))
	params := reorgParams{Index: count, Message: newMessages, MessageForPrefetch: oldMessages}
	var result []*execution.MessageResult
	if err := c.client.CallContext(ctx, &result, "Reorg", params); err != nil {
		log.Error("Failed to call Reorg", "error", err)
		return nil, fmt.Errorf("failed to call Reorg: %w", err)
	}
	return result, nil
}

func (c *nethRpcClient) SequenceDelayedMessage(ctx context.Context, message *arbostypes.L1IncomingMessage, delayedSeqNum uint64) error {
	log.Info("Making JSON-RPC call to SequenceDelayedMessage", "url", c.url, "delayedSeqNum", delayedSeqNum)
	params := seqDelayedParams{DelayedSeqNum: delayedSeqNum, Message: message}
	var result execution.MessageResult
	if err := c.client.CallContext(ctx, &result, "SequenceDelayedMessage", params); err != nil {
		log.Error("Failed to call SequenceDelayedMessage", "error", err)
		return fmt.Errorf("failed to call SequenceDelayedMessage: %w", err)
	}
	return nil
}
