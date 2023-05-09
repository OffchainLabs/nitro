// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbnode

var (
	BlockValidatorPrefix       string = "v"         // the prefix for all block validator keys
	messagePrefix              []byte = []byte("m") // maps a message sequence number to a message
	legacyDelayedMessagePrefix []byte = []byte("d") // maps a delayed sequence number to an accumulator and a message as serialized on L1
	rlpDelayedMessagePrefix    []byte = []byte("e") // maps a delayed sequence number to an accumulator and an RLP encoded message
	sequencerBatchMetaPrefix   []byte = []byte("s") // maps a batch sequence number to BatchMetadata
	delayedSequencedPrefix     []byte = []byte("a") // maps a delayed message count to the first sequencer batch sequence number with this delayed count

	messageCountKey                 []byte = []byte("_messageCount")                 // contains the current message count
	delayedMessageCountKey          []byte = []byte("_delayedMessageCount")          // contains the current delayed message count
	sequencerBatchCountKey          []byte = []byte("_sequencerBatchCount")          // contains the current sequencer message count
	dbSchemaVersion                 []byte = []byte("_schemaVersion")                // contains a uint64 representing the database schema version
	lastPrunedSequencerBatchMetaKey []byte = []byte("_lastPrunedSequencerBatchMeta") // contains the last pruned batch metadata key
)

const currentDbSchemaVersion uint64 = 1
