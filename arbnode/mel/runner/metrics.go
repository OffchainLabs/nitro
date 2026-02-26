package melrunner

import "github.com/ethereum/go-ethereum/metrics"

var (
	// FSM health.
	stuckFSMIndicatingGauge   = metrics.NewRegisteredGauge("arb/mel/stuck", nil) // 1-stuck, 0-not_stuck
	fsmBlocksProcessedCounter = metrics.NewRegisteredCounter("arb/mel/fsm/process_block", nil)
	fsmReorgsCounter          = metrics.NewRegisteredCounter("arb/mel/fsm/reorging", nil)
	fsmSaveMessagesCounter    = metrics.NewRegisteredCounter("arb/mel/fsm/save_messages", nil)

	// State progress.
	latestBlockGauge            = metrics.NewRegisteredGauge("arb/mel/latest/parent_chain_block_number", nil)
	latestMsgCountGauge         = metrics.NewRegisteredGauge("arb/mel/latest/msg_count", nil)
	latestDelayedSeenCountGauge = metrics.NewRegisteredGauge("arb/mel/latest/delayed_msg_seen_count", nil)
	latestDelayedReadCountGauge = metrics.NewRegisteredGauge("arb/mel/latest/delayed_msg_read_count", nil)

	// Throughput.
	msgsExtractedCounter = metrics.NewRegisteredCounter("arb/mel/msgs/extracted", nil)
	msgsPushedCounter    = metrics.NewRegisteredCounter("arb/mel/msgs/pushed_to_execution", nil)

	// Errors.
	extractionErrors = metrics.NewRegisteredCounter("arb/mel/errors/extraction_function_errors", nil)

	// Reorgs
	reorgCounter = metrics.NewRegisteredCounter("arb/mel/reorgs", nil)

	// Performance.
	blockProcessTimeGauge = metrics.NewRegisteredGauge("arb/mel/block_processing_time_millis", nil)
)
