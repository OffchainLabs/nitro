// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbnode

import "github.com/offchainlabs/nitro/arbnode/db-schema"

var (
	messagePrefix                       = dbschema.MessagePrefix
	blockHashInputFeedPrefix            = dbschema.BlockHashInputFeedPrefix
	blockMetadataInputFeedPrefix        = dbschema.BlockMetadataInputFeedPrefix
	missingBlockMetadataInputFeedPrefix = dbschema.MissingBlockMetadataInputFeedPrefix
	messageResultPrefix                 = dbschema.MessageResultPrefix
	legacyDelayedMessagePrefix          = dbschema.LegacyDelayedMessagePrefix
	rlpDelayedMessagePrefix             = dbschema.RlpDelayedMessagePrefix
	parentChainBlockNumberPrefix        = dbschema.ParentChainBlockNumberPrefix
	sequencerBatchMetaPrefix            = dbschema.SequencerBatchMetaPrefix
	delayedSequencedPrefix              = dbschema.DelayedSequencedPrefix

	messageCountKey             = dbschema.MessageCountKey
	lastPrunedMessageKey        = dbschema.LastPrunedMessageKey
	lastPrunedDelayedMessageKey = dbschema.LastPrunedDelayedMessageKey
	delayedMessageCountKey      = dbschema.DelayedMessageCountKey
	sequencerBatchCountKey      = dbschema.SequencerBatchCountKey
	dbSchemaVersion             = dbschema.DbSchemaVersion
)

const currentDbSchemaVersion uint64 = dbschema.CurrentDbSchemaVersion
