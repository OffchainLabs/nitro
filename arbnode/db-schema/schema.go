// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package dbschema

var (
	MessagePrefix                       []byte = []byte("m") // maps a message sequence number to a message
	BlockHashInputFeedPrefix            []byte = []byte("b") // maps a message sequence number to a block hash received through the input feed
	BlockMetadataInputFeedPrefix        []byte = []byte("t") // maps a message sequence number to a blockMetaData byte array received through the input feed
	MissingBlockMetadataInputFeedPrefix []byte = []byte("x") // maps a message sequence number whose blockMetaData byte array is missing to nil
	MessageResultPrefix                 []byte = []byte("r") // maps a message sequence number to a message result
	LegacyDelayedMessagePrefix          []byte = []byte("d") // maps a delayed sequence number to an accumulator and a message as serialized on L1
	RlpDelayedMessagePrefix             []byte = []byte("e") // maps a delayed sequence number to an accumulator and an RLP encoded message
	ParentChainBlockNumberPrefix        []byte = []byte("p") // maps a delayed sequence number to a parent chain block number
	SequencerBatchMetaPrefix            []byte = []byte("s") // maps a batch sequence number to BatchMetadata
	DelayedSequencedPrefix              []byte = []byte("a") // maps a delayed message count to the first sequencer batch sequence number with this delayed count
	MelStatePrefix                      []byte = []byte("l") // maps a parent chain block number to its computed MEL state
	MelDelayedMessagePrefix             []byte = []byte("y") // maps a delayed sequence number to an accumulator and an RLP encoded message [Note: might need to replace or be replaced by RlpDelayedMessagePrefix]

	MessageCountKey             []byte = []byte("_messageCount")                // contains the current message count
	LastPrunedMessageKey        []byte = []byte("_lastPrunedMessageKey")        // contains the last pruned message key
	LastPrunedDelayedMessageKey []byte = []byte("_lastPrunedDelayedMessageKey") // contains the last pruned RLP delayed message key
	DelayedMessageCountKey      []byte = []byte("_delayedMessageCount")         // contains the current delayed message count
	SequencerBatchCountKey      []byte = []byte("_sequencerBatchCount")         // contains the current sequencer message count
	DbSchemaVersion             []byte = []byte("_schemaVersion")               // contains a uint64 representing the database schema version
	HeadMelStateBlockNumKey     []byte = []byte("_headMelStateBlockNum")        // contains the latest computed MEL state's parent chain block number
)

const CurrentDbSchemaVersion uint64 = 2
