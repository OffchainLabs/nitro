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
	fatalErrChan              chan error
}

func NewCompareExecutionClient(gethExecutionClient *gethexec.ExecutionNode, nethermindExecutionClient *nethermindExecutionClient, fatalErrChan chan error) *compareExecutionClient {
	return &compareExecutionClient{
		gethExecutionClient:       gethExecutionClient,
		nethermindExecutionClient: nethermindExecutionClient,
		fatalErrChan:              fatalErrChan,
	}
}

func comparePromises[T any](fatalErrChan chan error, op string,
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
			select {
			case fatalErrChan <- fmt.Errorf("compareExecutionClient %s: %s", op, err.Error()):
				// Successfully sent - this is a fatal operation
				promise.ProduceError(err)
			default:
				// Could not send (nil channel or full) - treat as non-fatal
				log.Error("Non-fatal comparison error", "operation", op, "err", err)
				promise.Produce(intRes)
			}
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
		return fmt.Errorf("internal operation failed: %v", intErr)
	case intErr == nil && extErr != nil:
		return fmt.Errorf("external operation failed: %v", extErr)
	default:
		if !cmp.Equal(intRes, extRes) {
			opts := cmp.Options{
				cmp.Transformer("HashHex", func(h common.Hash) string { return h.Hex() }),
			}
			diff := cmp.Diff(intRes, extRes, opts)
			// Log the detailed diff using fmt.Printf to avoid escaping
			fmt.Printf("ERROR: Execution mismatch detected in operation: %s\n", op)
			fmt.Printf("Diff details:\n%s\n", diff)
			return fmt.Errorf("execution mismatch in %s", op)
		}
	}
	return nil
}

func (w *compareExecutionClient) DigestMessage(index arbutil.MessageIndex, msg *arbostypes.MessageWithMetadata, msgForPrefetch *arbostypes.MessageWithMetadata) containers.PromiseInterface[*execution.MessageResult] {
	start := time.Now()
	log.Info("CompareExecutionClient: DigestMessage", "index", index)
	internal := w.gethExecutionClient.DigestMessage(index, msg, msgForPrefetch)
	external := w.nethermindExecutionClient.DigestMessage(index, msg, msgForPrefetch)

	result := comparePromises(w.fatalErrChan,
		"DigestMessage",
		internal,
		external,
	)
	log.Info("CompareExecutionClient: DigestMessage completed", "index", index, "elapsed", time.Since(start))
	return result
}

func (w *compareExecutionClient) Reorg(count arbutil.MessageIndex, newMessages []arbostypes.MessageWithMetadataAndBlockInfo, oldMessages []*arbostypes.MessageWithMetadata) containers.PromiseInterface[[]*execution.MessageResult] {
	start := time.Now()
	log.Info("CompareExecutionClient: Reorg", "count", count, "newMessagesCount", len(newMessages), "oldMessagesCount", len(oldMessages))

	internal := w.gethExecutionClient.Reorg(count, newMessages, oldMessages)
	external := w.nethermindExecutionClient.Reorg(count, newMessages, oldMessages)

	result := comparePromises(w.fatalErrChan, "Reorg", internal, external)
	log.Info("CompareExecutionClient: Reorg completed", "count", count, "elapsed", time.Since(start))
	return result
}

func (w *compareExecutionClient) HeadMessageIndex() containers.PromiseInterface[arbutil.MessageIndex] {
	start := time.Now()
	log.Info("CompareExecutionClient: HeadMessageIndex")
	internal := w.gethExecutionClient.HeadMessageIndex()
	external := w.nethermindExecutionClient.HeadMessageIndex()
	result := comparePromises(nil, "HeadMessageIndex", internal, external)
	log.Info("CompareExecutionClient: HeadMessageIndex completed", "elapsed", time.Since(start))
	return result
}

func (w *compareExecutionClient) ResultAtMessageIndex(index arbutil.MessageIndex) containers.PromiseInterface[*execution.MessageResult] {
	start := time.Now()
	log.Info("CompareExecutionClient: ResultAtMessageIndex", "index", index)
	internal := w.gethExecutionClient.ResultAtMessageIndex(index)
	external := w.nethermindExecutionClient.ResultAtMessageIndex(index)
	result := comparePromises(nil, "ResultAtMessageIndex", internal, external)
	log.Info("CompareExecutionClient: ResultAtMessageIndex completed", "index", index, "elapsed", time.Since(start))
	return result
}

func (w *compareExecutionClient) MessageIndexToBlockNumber(messageIndex arbutil.MessageIndex) containers.PromiseInterface[uint64] {
	start := time.Now()
	log.Info("CompareExecutionClient: MessageIndexToBlockNumber", "messageIndex", messageIndex)
	internal := w.gethExecutionClient.MessageIndexToBlockNumber(messageIndex)
	external := w.nethermindExecutionClient.MessageIndexToBlockNumber(messageIndex)
	result := comparePromises(w.fatalErrChan, "MessageIndexToBlockNumber", internal, external)
	log.Info("CompareExecutionClient: MessageIndexToBlockNumber completed", "messageIndex", messageIndex, "elapsed", time.Since(start))
	return result
}

func (w *compareExecutionClient) BlockNumberToMessageIndex(blockNum uint64) containers.PromiseInterface[arbutil.MessageIndex] {
	start := time.Now()
	log.Info("CompareExecutionClient: BlockNumberToMessageIndex", "blockNum", blockNum)
	internal := w.gethExecutionClient.BlockNumberToMessageIndex(blockNum)
	external := w.nethermindExecutionClient.BlockNumberToMessageIndex(blockNum)
	result := comparePromises(w.fatalErrChan, "BlockNumberToMessageIndex", internal, external)
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
	return comparePromises(w.fatalErrChan, "SetFinalityData", internal, external)
}

func (w *compareExecutionClient) MarkFeedStart(to arbutil.MessageIndex) containers.PromiseInterface[struct{}] {
	start := time.Now()
	log.Info("CompareExecutionClient: MarkFeedStart", "to", to)
	internal := w.gethExecutionClient.MarkFeedStart(to)
	external := w.nethermindExecutionClient.MarkFeedStart(to)
	result := comparePromises(w.fatalErrChan, "MarkFeedStart", internal, external)
	log.Info("CompareExecutionClient: MarkFeedStart completed", "to", to, "elapsed", time.Since(start))
	return result
}

func (w *compareExecutionClient) TriggerMaintenance() containers.PromiseInterface[struct{}] {
	start := time.Now()
	log.Info("CompareExecutionClient: TriggerMaintenance")
	result := w.gethExecutionClient.TriggerMaintenance()
	log.Info("CompareExecutionClient: TriggerMaintenance completed", "elapsed", time.Since(start))
	return result
}

func (w *compareExecutionClient) ShouldTriggerMaintenance() containers.PromiseInterface[bool] {
	start := time.Now()
	log.Info("CompareExecutionClient: ShouldTriggerMaintenance")
	internal := w.gethExecutionClient.ShouldTriggerMaintenance()
	external := w.nethermindExecutionClient.ShouldTriggerMaintenance()
	result := comparePromises(w.fatalErrChan, "ShouldTriggerMaintenance", internal, external)
	log.Info("CompareExecutionClient: ShouldTriggerMaintenance completed", "elapsed", time.Since(start))
	return result
}

func (w *compareExecutionClient) MaintenanceStatus() containers.PromiseInterface[*execution.MaintenanceStatus] {
	start := time.Now()
	log.Info("CompareExecutionClient: MaintenanceStatus")
	internal := w.gethExecutionClient.MaintenanceStatus()
	external := w.nethermindExecutionClient.MaintenanceStatus()
	result := comparePromises(w.fatalErrChan, "MaintenanceStatus", internal, external)
	log.Info("CompareExecutionClient: MaintenanceStatus completed", "elapsed", time.Since(start))
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

	if err := compare("SequenceDelayedMessage", struct{}{}, internalErr, struct{}{}, externalErr); err != nil {
		// Send to fatal error channel for graceful shutdown
		select {
		case w.fatalErrChan <- fmt.Errorf("compareExecutionClient SequenceDelayedMessage: %s", err.Error()):
		default:
			log.Error("Failed to send comparison error to fatal channel", "err", err)
		}

		return err
	}

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

func (w *compareExecutionClient) RecordBlockCreation(ctx context.Context, index arbutil.MessageIndex, msg *arbostypes.MessageWithMetadata) (*execution.RecordResult, error) {
	start := time.Now()
	log.Info("CompareExecutionClient: RecordBlockCreation", "index", index)
	result, err := w.gethExecutionClient.RecordBlockCreation(ctx, index, msg)
	log.Info("CompareExecutionClient: RecordBlockCreation completed", "index", index, "err", err, "elapsed", time.Since(start))
	return result, err
}

func (w *compareExecutionClient) MarkValid(index arbutil.MessageIndex, resultHash common.Hash) {
	start := time.Now()
	log.Info("CompareExecutionClient: MarkValid", "index", index, "resultHash", resultHash)
	w.gethExecutionClient.MarkValid(index, resultHash)
	log.Info("CompareExecutionClient: MarkValid completed", "index", index, "elapsed", time.Since(start))
}

func (w *compareExecutionClient) PrepareForRecord(ctx context.Context, start, end arbutil.MessageIndex) error {
	startTime := time.Now()
	log.Info("CompareExecutionClient: PrepareForRecord", "start", start, "end", end)
	err := w.gethExecutionClient.PrepareForRecord(ctx, start, end)
	log.Info("CompareExecutionClient: PrepareForRecord completed", "start", start, "end", end, "err", err, "elapsed", time.Since(startTime))
	return err
}

// ---- execution.ExecutionBatchindexter interface methods ----

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
