//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbnode

var (
	arbitrumPrefix            string = "\t"        // the prefix for all Arbitrum specific keys
	messageCountToBlockPrefix []byte = []byte("b") // maps a message count to a block
	messagePrefix             []byte = []byte("m") // maps a message sequence number to a message
	delayedMessagePrefix      []byte = []byte("d") // maps a delayed sequence number to an accumulator and a message
	sequencerBatchMetaPrefix  []byte = []byte("s") // maps a batch sequence number to an accumulator
	delayedSequencedPrefix    []byte = []byte("a") // maps a delayed message count to a sequencer batch sequence number

	messageCountKey        []byte = []byte("_messageCount")        // contains the current message count
	delayedMessageCountKey []byte = []byte("_delayedMessageCount") // contains the current delayed message count
	sequencerBatchCountKey []byte = []byte("_sequencerBatchCount") // contains the current sequencer message count
)
