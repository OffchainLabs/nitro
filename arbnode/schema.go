// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbnode

type databaseColumn struct {
	Prefix               []byte
	keyLengthAfterPrefix int
}

func (c *databaseColumn) KeyLength() int {
	fullLen := len(c.Prefix) + c.keyLengthAfterPrefix
	if fullLen+len(arbitrumPrefix) == 32 {
		panic("Database column with prefix " + string(c.Prefix) + " has the same key length as a hash")
	}
	return fullLen
}

var (
	arbitrumPrefix       = "\t"                 // the prefix for all Arbitrum specific keys
	blockValidatorPrefix = arbitrumPrefix + "v" // the prefix for all block validator keys

	messageColumn            = databaseColumn{[]byte("m"), 8} // maps a message sequence number to a message
	delayedMessageColumn     = databaseColumn{[]byte("d"), 8} // maps a delayed sequence number to an accumulator and a message
	sequencerBatchMetaColumn = databaseColumn{[]byte("s"), 8} // maps a batch sequence number to BatchMetadata
	delayedSequencedColumn   = databaseColumn{[]byte("a"), 8} // maps a delayed message count to the first sequencer batch sequence number with this delayed count

	messageCountKey        []byte = []byte("_messageCount")        // contains the current message count
	delayedMessageCountKey []byte = []byte("_delayedMessageCount") // contains the current delayed message count
	sequencerBatchCountKey []byte = []byte("_sequencerBatchCount") // contains the current sequencer message count
)
