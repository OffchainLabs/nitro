package execution_engine

import (
	"context"
	"fmt"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/util/containers"
)

// ErrorExecutionClientUnimplementedMethod is an error type that is returned
// when an unimplemented method of the execution.ExecutionClient interface
// is invoked.
type ErrorExecutionClientUnimplementedMethod struct {
	Method string
}

// Error implements error
func (e ErrorExecutionClientUnimplementedMethod) Error() string {
	return fmt.Sprintf("unimplemented method for execution client: %s", e.Method)
}

// UnimplementedExecutionClient is an implementation of
// execution.ExecutionClient with all methods resulting in a panic
// on invocation.
//
// This is useful, as it allows future implementors to create simpler
// mocks by falling back on this implementation, rather than
// implementing all methods of the interface.
type UnimplementedExecutionClient struct{}

// Compile time check to ensure that UnimplementedExecutionClient implements
// the execution.ExecutionClient interface.
var _ execution.ExecutionClient = UnimplementedExecutionClient{}

// BlockNumberToMessageIndex implements execution.ExecutionClient.
func (u UnimplementedExecutionClient) BlockNumberToMessageIndex(blockNum uint64) containers.PromiseInterface[arbutil.MessageIndex] {
	panic(ErrorExecutionClientUnimplementedMethod{"BlockNumberToMessageIndex"})
}

// HeadMessageIndex implements execution.ExecutionClient.
func (u UnimplementedExecutionClient) HeadMessageIndex() containers.PromiseInterface[arbutil.MessageIndex] {
	panic(ErrorExecutionClientUnimplementedMethod{"HeadMessageIndex"})
}

// Maintenance implements execution.ExecutionClient.
func (u UnimplementedExecutionClient) Maintenance() containers.PromiseInterface[struct{}] {
	panic(ErrorExecutionClientUnimplementedMethod{"Maintenance"})
}

// MarkFeedStart implements execution.ExecutionClient.
func (u UnimplementedExecutionClient) MarkFeedStart(to arbutil.MessageIndex) containers.PromiseInterface[struct{}] {
	panic(ErrorExecutionClientUnimplementedMethod{"MarkFeedStart"})
}

// MessageIndexToBlockNumber implements execution.ExecutionClient.
func (u UnimplementedExecutionClient) MessageIndexToBlockNumber(messageNum arbutil.MessageIndex) containers.PromiseInterface[uint64] {
	panic(ErrorExecutionClientUnimplementedMethod{"MessageIndexToBlockNumber"})
}

// Reorg implements execution.ExecutionClient.
func (u UnimplementedExecutionClient) Reorg(msgIdxOfFirstMsgToAdd arbutil.MessageIndex, newMessages []arbostypes.MessageWithMetadataAndBlockInfo, oldMessages []*arbostypes.MessageWithMetadata) containers.PromiseInterface[[]*execution.MessageResult] {
	panic(ErrorExecutionClientUnimplementedMethod{"Reorg"})
}

// ResultAtMessageIndex implements execution.ExecutionClient.
func (u UnimplementedExecutionClient) ResultAtMessageIndex(msgIdx arbutil.MessageIndex) containers.PromiseInterface[*execution.MessageResult] {
	panic(ErrorExecutionClientUnimplementedMethod{"ResultAtMessageIndex"})
}

// SetFinalityData implements execution.ExecutionClient.
func (u UnimplementedExecutionClient) SetFinalityData(ctx context.Context, safeFinalityData *arbutil.FinalityData, finalizedFinalityData *arbutil.FinalityData, validatedFinalityData *arbutil.FinalityData) containers.PromiseInterface[struct{}] {
	panic(ErrorExecutionClientUnimplementedMethod{"SetFinalityData"})
}

// Start implements execution.ExecutionClient.
func (u UnimplementedExecutionClient) Start(ctx context.Context) error {
	panic(ErrorExecutionClientUnimplementedMethod{"Start"})
}

// StopAndWait implements execution.ExecutionClient.
func (u UnimplementedExecutionClient) StopAndWait() {
	panic(ErrorExecutionClientUnimplementedMethod{"StopAndWait"})
}

// DigestMessage implements execution.ExecutionClient
// This
func (UnimplementedExecutionClient) DigestMessage(msgIdx arbutil.MessageIndex, msg *arbostypes.MessageWithMetadata, msgForPrefetch *arbostypes.MessageWithMetadata) containers.PromiseInterface[*execution.MessageResult] {
	panic(ErrorExecutionClientUnimplementedMethod{"DigestMessage"})
}
