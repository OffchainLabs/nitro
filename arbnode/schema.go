// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbnode

import (
	"github.com/offchainlabs/nitro/arbnode/db/schema"
)

var (
	messagePrefix                       = schema.MessagePrefix
	blockHashInputFeedPrefix            = schema.BlockHashInputFeedPrefix
	blockMetadataInputFeedPrefix        = schema.BlockMetadataInputFeedPrefix
	missingBlockMetadataInputFeedPrefix = schema.MissingBlockMetadataInputFeedPrefix
	messageResultPrefix                 = schema.MessageResultPrefix
	legacyDelayedMessagePrefix          = schema.LegacyDelayedMessagePrefix
	rlpDelayedMessagePrefix             = schema.RlpDelayedMessagePrefix
	parentChainBlockNumberPrefix        = schema.ParentChainBlockNumberPrefix
	sequencerBatchMetaPrefix            = schema.SequencerBatchMetaPrefix
	delayedSequencedPrefix              = schema.DelayedSequencedPrefix

	messageCountKey             = schema.MessageCountKey
	lastPrunedMessageKey        = schema.LastPrunedMessageKey
	lastPrunedDelayedMessageKey = schema.LastPrunedDelayedMessageKey
	delayedMessageCountKey      = schema.DelayedMessageCountKey
	sequencerBatchCountKey      = schema.SequencerBatchCountKey
	dbSchemaVersion             = schema.DbSchemaVersion
)

const currentDbSchemaVersion uint64 = schema.CurrentDbSchemaVersion
