package melrunner

import "github.com/ethereum/go-ethereum/metrics"

var (
	// FSM health.
	stuckFSMIndicatingGauge   = metrics.NewRegisteredGauge("arb/mel/stuck", nil) // 1-stuck, 0-not_stuck
	fsmBlocksProcessedCounter = metrics.NewRegisteredCounter("arb/mel/fsm/process_block_total", nil)
	fsmSaveMessagesCounter    = metrics.NewRegisteredCounter("arb/mel/fsm/save_messages_total", nil)

	// State progress.
	latestBlockGauge            = metrics.NewRegisteredGauge("arb/mel/latest/parent_chain_block_number", nil)
	latestMsgCountGauge         = metrics.NewRegisteredGauge("arb/mel/latest/msg_count", nil)
	latestDelayedSeenCountGauge = metrics.NewRegisteredGauge("arb/mel/latest/delayed_msg_seen_count", nil)
	latestDelayedReadCountGauge = metrics.NewRegisteredGauge("arb/mel/latest/delayed_msg_read_count", nil)

	// Throughput.
	msgsExtractedCounter = metrics.NewRegisteredCounter("arb/mel/msgs/extracted_total", nil)
	msgsPushedCounter    = metrics.NewRegisteredCounter("arb/mel/msgs/pushed_to_execution_total", nil)

	// Errors.
	extractionErrors = metrics.NewRegisteredCounter("arb/mel/errors/extraction_function_errors_total", nil)

	// Reorgs
	reorgCounter = metrics.NewRegisteredCounter("arb/mel/reorgs_total", nil)

	// Performance.
	blockProcessTimeGauge = metrics.NewRegisteredGauge("arb/mel/block_processing_time_micros", nil)

	// MEL state size bytes.
	melStateSizeBytesGauge = metrics.NewRegisteredGauge("arb/mel/mel_state_size_bytes", nil)
)
