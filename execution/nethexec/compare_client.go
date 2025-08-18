package nethexec

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/google/go-cmp/cmp"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/util/containers"
)

type FullExecutionClient interface {
	execution.ExecutionSequencer // includes ExecutionClient
	execution.ExecutionRecorder
	execution.ExecutionBatchPoster
}

var (
	_ FullExecutionClient         = (*compareExecutionClient)(nil)
	_ arbnode.ExecutionNodeBridge = (*compareExecutionClient)(nil)
)

type compareExecutionClient struct {
	gethExecutionClient       *gethexec.ExecutionNode
	nethermindExecutionClient *nethermindExecutionClient
}

func NewCompareExecutionClient(gethExecutionClient *gethexec.ExecutionNode, nethermindExecutionClient *nethermindExecutionClient) *compareExecutionClient {
	return &compareExecutionClient{
		gethExecutionClient:       gethExecutionClient,
		nethermindExecutionClient: nethermindExecutionClient,
	}
}

func comparePromises[T any](op string,
	internal containers.PromiseInterface[T],
	external containers.PromiseInterface[T],
) containers.PromiseInterface[T] {
	promise := containers.NewPromise[T](nil)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		intRes, intErr := internal.Await(ctx)
		extRes, extErr := external.Await(ctx)

		if err := compare(op, intRes, intErr, extRes, extErr); err != nil {
			promise.ProduceError(err)
		} else {
			promise.Produce(intRes)
		}
	}()
	return &promise
}

func compare[T any](op string, intRes T, intErr error, extRes T, extErr error) error {
	switch {
	case intErr != nil && extErr != nil:
		return fmt.Errorf("both operations failed: internal=%v external=%v", intErr, extErr)
	case intErr != nil && extErr == nil:
		panic(fmt.Sprintf("internal operation failed: %v", intErr))
	case intErr == nil && extErr != nil:
		panic(fmt.Sprintf("external operation failed: %v", extErr))
	default:
		if !cmp.Equal(intRes, extRes) {
			opts := cmp.Options{
				cmp.Transformer("HashHex", func(h common.Hash) string { return h.Hex() }),
			}
			diff := cmp.Diff(intRes, extRes, opts)
			panic(fmt.Sprintf("Execution mismatch between internal and external:\n%s\n%s", op, diff))
		}
	}
	return nil
}

func (w *compareExecutionClient) DigestMessage(num arbutil.MessageIndex, msg *arbostypes.MessageWithMetadata, msgForPrefetch *arbostypes.MessageWithMetadata) containers.PromiseInterface[*execution.MessageResult] {
	start := time.Now()
	log.Info("CompareExecutionClient: DigestMessage", "num", num)
	internal := w.gethExecutionClient.DigestMessage(num, msg, msgForPrefetch)
	external := w.nethermindExecutionClient.DigestMessage(num, msg, msgForPrefetch)

	result := comparePromises(
		"DigestMessage",
		internal,
		external,
	)
	log.Info("CompareExecutionClient: DigestMessage completed", "num", num, "elapsed", time.Since(start))
	return result
}

func (w *compareExecutionClient) Reorg(count arbutil.MessageIndex, newMessages []arbostypes.MessageWithMetadataAndBlockInfo, oldMessages []*arbostypes.MessageWithMetadata) containers.PromiseInterface[[]*execution.MessageResult] {
	start := time.Now()
	log.Info("CompareExecutionClient: Reorg", "count", count, "newMessagesCount", len(newMessages), "oldMessagesCount", len(oldMessages))

	internal := w.gethExecutionClient.Reorg(count, newMessages, oldMessages)
	external := w.nethermindExecutionClient.Reorg(count, newMessages, oldMessages)

	result := comparePromises("Reorg", internal, external)
	log.Info("CompareExecutionClient: Reorg completed", "count", count, "elapsed", time.Since(start))
	return result
}

func (w *compareExecutionClient) HeadMessageIndex() containers.PromiseInterface[arbutil.MessageIndex] {
	start := time.Now()
	log.Info("CompareExecutionClient: HeadMessageIndex")
	internal := w.gethExecutionClient.HeadMessageIndex()
	external := w.nethermindExecutionClient.HeadMessageIndex()
	result := comparePromises("HeadMessageIndex", internal, external)
	log.Info("CompareExecutionClient: HeadMessageIndex completed", "elapsed", time.Since(start))
	return result
}

func (w *compareExecutionClient) ResultAtMessageIndex(pos arbutil.MessageIndex) containers.PromiseInterface[*execution.MessageResult] {
	start := time.Now()
	log.Info("CompareExecutionClient: ResultAtMessageIndex", "pos", pos)
	internal := w.gethExecutionClient.ResultAtMessageIndex(pos)
	external := w.nethermindExecutionClient.ResultAtMessageIndex(pos)
	result := comparePromises("ResultAtMessageIndex", internal, external)
	log.Info("CompareExecutionClient: ResultAtMessageIndex completed", "pos", pos, "elapsed", time.Since(start))
	return result
}

func (w *compareExecutionClient) MessageIndexToBlockNumber(messageNum arbutil.MessageIndex) containers.PromiseInterface[uint64] {
	start := time.Now()
	log.Info("CompareExecutionClient: MessageIndexToBlockNumber", "messageNum", messageNum)
	internal := w.gethExecutionClient.MessageIndexToBlockNumber(messageNum)
	external := w.nethermindExecutionClient.MessageIndexToBlockNumber(messageNum)
	result := comparePromises("MessageIndexToBlockNumber", internal, external)
	log.Info("CompareExecutionClient: MessageIndexToBlockNumber completed", "messageNum", messageNum, "elapsed", time.Since(start))
	return result
}

func (w *compareExecutionClient) BlockNumberToMessageIndex(blockNum uint64) containers.PromiseInterface[arbutil.MessageIndex] {
	start := time.Now()
	log.Info("CompareExecutionClient: BlockNumberToMessageIndex", "blockNum", blockNum)
	internal := w.gethExecutionClient.BlockNumberToMessageIndex(blockNum)
	external := w.nethermindExecutionClient.BlockNumberToMessageIndex(blockNum)
	result := comparePromises("BlockNumberToMessageIndex", internal, external)
	log.Info("CompareExecutionClient: BlockNumberToMessageIndex completed", "blockNum", blockNum, "elapsed", time.Since(start))
	return result
}

func (w *compareExecutionClient) SetFinalityData(ctx context.Context, finalityData *arbutil.FinalityData, finalizedFinalityData *arbutil.FinalityData, validatedFinalityData *arbutil.FinalityData) containers.PromiseInterface[struct{}] {
	log.Info("CompareExecutionClient: SetFinalityData",
		"safeFinalityData", finalityData,
		"finalizedFinalityData", finalizedFinalityData,
		"validatedFinalityData", validatedFinalityData)

	internal := w.gethExecutionClient.SetFinalityData(ctx, finalityData, finalizedFinalityData, validatedFinalityData)
	external := w.nethermindExecutionClient.SetFinalityData(ctx, finalityData, finalizedFinalityData, validatedFinalityData)
	return comparePromises("SetFinalityData", internal, external)
}

func (w *compareExecutionClient) MarkFeedStart(to arbutil.MessageIndex) containers.PromiseInterface[struct{}] {
	start := time.Now()
	log.Info("CompareExecutionClient: MarkFeedStart", "to", to)
	internal := w.gethExecutionClient.MarkFeedStart(to)
	external := w.nethermindExecutionClient.MarkFeedStart(to)
	result := comparePromises("MarkFeedStart", internal, external)
	log.Info("CompareExecutionClient: MarkFeedStart completed", "to", to, "elapsed", time.Since(start))
	return result
}

func (w *compareExecutionClient) Maintenance() containers.PromiseInterface[struct{}] {
	start := time.Now()
	log.Info("CompareExecutionClient: Maintenance")
	result := w.gethExecutionClient.Maintenance()
	log.Info("CompareExecutionClient: Maintenance completed", "elapsed", time.Since(start))
	return result
}

func (w *compareExecutionClient) Start(ctx context.Context) error {
	start := time.Now()
	log.Info("CompareExecutionClient: Start")
	err := w.gethExecutionClient.Start(ctx)
	log.Info("CompareExecutionClient: Start completed", "elapsed", time.Since(start))
	return err
}

func (w *compareExecutionClient) StopAndWait() {
	start := time.Now()
	log.Info("CompareExecutionClient: StopAndWait")
	w.gethExecutionClient.StopAndWait()
	log.Info("CompareExecutionClient: StopAndWait completed", "elapsed", time.Since(start))
}

// ---- execution.ExecutionSequencer interface methods ----

func (w *compareExecutionClient) Pause() {
	start := time.Now()
	log.Info("CompareExecutionClient: Pause")
	w.gethExecutionClient.Pause()
	log.Info("CompareExecutionClient: Pause completed", "elapsed", time.Since(start))
}

func (w *compareExecutionClient) Activate() {
	start := time.Now()
	log.Info("CompareExecutionClient: Activate")
	w.gethExecutionClient.Activate()
	log.Info("CompareExecutionClient: Activate completed", "elapsed", time.Since(start))
}

func (w *compareExecutionClient) ForwardTo(url string) error {
	start := time.Now()
	log.Info("CompareExecutionClient: ForwardTo", "url", url)
	err := w.gethExecutionClient.ForwardTo(url)
	log.Info("CompareExecutionClient: ForwardTo completed", "url", url, "err", err, "elapsed", time.Since(start))
	return err
}

func (w *compareExecutionClient) SequenceDelayedMessage(message *arbostypes.L1IncomingMessage, delayedSeqNum uint64) error {
	start := time.Now()
	log.Info("CompareExecutionClient: SequenceDelayedMessage", "delayedSeqNum", delayedSeqNum)

	internalErr := w.gethExecutionClient.SequenceDelayedMessage(message, delayedSeqNum)
	externalErr := w.nethermindExecutionClient.SequenceDelayedMessage(message, delayedSeqNum)

	compare("SequenceDelayedMessage", struct{}{}, internalErr, struct{}{}, externalErr)

	log.Info("CompareExecutionClient: SequenceDelayedMessage completed", "delayedSeqNum", delayedSeqNum, "err", internalErr, "elapsed", time.Since(start))
	return internalErr
}

func (w *compareExecutionClient) NextDelayedMessageNumber() (uint64, error) {
	// start := time.Now()
	// log.Info("CompareExecutionClient: NextDelayedMessageNumber")
	result, err := w.gethExecutionClient.NextDelayedMessageNumber()
	// log.Info("CompareExecutionClient: NextDelayedMessageNumber completed", "result", result, "err", err, "elapsed", time.Since(start))
	return result, err
}

func (w *compareExecutionClient) Synced(ctx context.Context) bool {
	start := time.Now()
	log.Info("CompareExecutionClient: Synced")
	result := w.gethExecutionClient.Synced(ctx)
	log.Info("CompareExecutionClient: Synced completed", "result", result, "elapsed", time.Since(start))
	return result
}

func (w *compareExecutionClient) FullSyncProgressMap(ctx context.Context) map[string]interface{} {
	start := time.Now()
	log.Info("CompareExecutionClient: FullSyncProgressMap")
	result := w.gethExecutionClient.FullSyncProgressMap(ctx)
	log.Info("CompareExecutionClient: FullSyncProgressMap completed", "elapsed", time.Since(start))
	return result
}

// ---- execution.ExecutionRecorder interface methods ----

func (w *compareExecutionClient) RecordBlockCreation(ctx context.Context, pos arbutil.MessageIndex, msg *arbostypes.MessageWithMetadata) (*execution.RecordResult, error) {
	start := time.Now()
	log.Info("CompareExecutionClient: RecordBlockCreation", "pos", pos)
	result, err := w.gethExecutionClient.RecordBlockCreation(ctx, pos, msg)
	log.Info("CompareExecutionClient: RecordBlockCreation completed", "pos", pos, "err", err, "elapsed", time.Since(start))
	return result, err
}

func (w *compareExecutionClient) MarkValid(pos arbutil.MessageIndex, resultHash common.Hash) {
	start := time.Now()
	log.Info("CompareExecutionClient: MarkValid", "pos", pos, "resultHash", resultHash)
	w.gethExecutionClient.MarkValid(pos, resultHash)
	log.Info("CompareExecutionClient: MarkValid completed", "pos", pos, "elapsed", time.Since(start))
}

func (w *compareExecutionClient) PrepareForRecord(ctx context.Context, start, end arbutil.MessageIndex) error {
	startTime := time.Now()
	log.Info("CompareExecutionClient: PrepareForRecord", "start", start, "end", end)
	err := w.gethExecutionClient.PrepareForRecord(ctx, start, end)
	log.Info("CompareExecutionClient: PrepareForRecord completed", "start", start, "end", end, "err", err, "elapsed", time.Since(startTime))
	return err
}

// ---- execution.ExecutionBatchPoster interface methods ----

func (w *compareExecutionClient) ArbOSVersionForMessageIndex(msgIdx arbutil.MessageIndex) (uint64, error) {
	start := time.Now()
	log.Info("CompareExecutionClient: ArbOSVersionForMessageIndex", "msgIdx", msgIdx)
	result, err := w.gethExecutionClient.ArbOSVersionForMessageIndex(msgIdx)
	log.Info("CompareExecutionClient: ArbOSVersionForMessageIndex completed", "msgIdx", msgIdx, "result", result, "err", err, "elapsed", time.Since(start))
	return result, err
}

func (w *compareExecutionClient) SetConsensusClient(consensus execution.FullConsensusClient) {
	w.gethExecutionClient.SetConsensusClient(consensus)
}

func (w *compareExecutionClient) Initialize(ctx context.Context) error {
	return w.gethExecutionClient.Initialize(ctx)
}
