// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package schema

import (
	"testing"
)

func TestPrefixUniqueness(t *testing.T) {
	// All single-byte DB key prefixes must be unique to prevent data corruption.
	prefixes := []struct {
		name  string
		value []byte
	}{
		{"MessagePrefix", MessagePrefix},
		{"BlockHashInputFeedPrefix", BlockHashInputFeedPrefix},
		{"BlockMetadataInputFeedPrefix", BlockMetadataInputFeedPrefix},
		{"MissingBlockMetadataInputFeedPrefix", MissingBlockMetadataInputFeedPrefix},
		{"MessageResultPrefix", MessageResultPrefix},
		{"LegacyDelayedMessagePrefix", LegacyDelayedMessagePrefix},
		{"RlpDelayedMessagePrefix", RlpDelayedMessagePrefix},
		{"ParentChainBlockNumberPrefix", ParentChainBlockNumberPrefix},
		{"SequencerBatchMetaPrefix", SequencerBatchMetaPrefix},
		{"DelayedSequencedPrefix", DelayedSequencedPrefix},
		{"MelStatePrefix", MelStatePrefix},
		{"MelDelayedMessagePrefix", MelDelayedMessagePrefix},
		{"MelSequencerBatchMetaPrefix", MelSequencerBatchMetaPrefix},
	}
	seen := make(map[string]string) // prefix string → variable name
	for _, p := range prefixes {
		key := string(p.value)
		if existing, ok := seen[key]; ok {
			t.Fatalf("prefix collision: %s and %s both use %q", existing, p.name, key)
		}
		seen[key] = p.name
	}

	keys := []struct {
		name  string
		value []byte
	}{
		{"MessageCountKey", MessageCountKey},
		{"LastPrunedMessageKey", LastPrunedMessageKey},
		{"LastPrunedDelayedMessageKey", LastPrunedDelayedMessageKey},
		{"LastPrunedLegacyDelayedMessageKey", LastPrunedLegacyDelayedMessageKey},
		{"LastPrunedMelDelayedMessageKey", LastPrunedMelDelayedMessageKey},
		{"LastPrunedParentChainBlockNumberKey", LastPrunedParentChainBlockNumberKey},
		{"DelayedMessageCountKey", DelayedMessageCountKey},
		{"SequencerBatchCountKey", SequencerBatchCountKey},
		{"DbSchemaVersion", DbSchemaVersion},
		{"HeadMelStateBlockNumKey", HeadMelStateBlockNumKey},
		{"InitialMelStateBlockNumKey", InitialMelStateBlockNumKey},
	}
	seenKeys := make(map[string]string)
	for _, k := range keys {
		key := string(k.value)
		if existing, ok := seenKeys[key]; ok {
			t.Fatalf("key collision: %s and %s both use %q", existing, k.name, key)
		}
		seenKeys[key] = k.name
	}
}
