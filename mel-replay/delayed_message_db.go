// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package melreplay

import (
	"bytes"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbnode/mel"
	"github.com/offchainlabs/nitro/arbnode/mel/extraction"
	"github.com/offchainlabs/nitro/arbutil"
)

type DelayedMessageDatabase struct {
	preimageResolver PreimageResolver
}

func NewDelayedMessageDatabase(preimageResolver PreimageResolver) melextraction.DelayedMessageDatabase {
	return &DelayedMessageDatabase{preimageResolver}
}

// ReadDelayedMessage pops the next delayed message from the outbox accumulator.
// If the outbox is empty, it pours the inbox into the outbox first using preimage resolution.
func (d *DelayedMessageDatabase) ReadDelayedMessage(
	state *mel.State,
	msgIndex uint64,
) (*mel.DelayedInboxMessage, error) {
	if msgIndex >= state.DelayedMessagesSeen {
		return nil, fmt.Errorf("index %d out of range, total delayed messages seen: %d", msgIndex, state.DelayedMessagesSeen)
	}
	// Pour inbox to outbox if outbox is empty
	if state.DelayedMessageOutboxAcc == (common.Hash{}) {
		if state.DelayedMessageInboxAcc == (common.Hash{}) {
			return nil, fmt.Errorf("both inbox and outbox are empty at index %d, cannot read delayed message", msgIndex)
		}
		if err := d.pourInboxToOutbox(state); err != nil {
			return nil, fmt.Errorf("error pouring delayed inbox to outbox: %w", err)
		}
	}
	// Pop from outbox
	result, err := d.preimageResolver.ResolveTypedPreimage(arbutil.Keccak256PreimageType, state.DelayedMessageOutboxAcc)
	if err != nil {
		return nil, fmt.Errorf("error resolving outbox preimage for delayed message at index %d: %w", msgIndex, err)
	}
	prevOutbox, msgHash, err := mel.SplitPreimage(result)
	if err != nil {
		return nil, fmt.Errorf("outbox preimage at index %d: %w", msgIndex, err)
	}
	state.DelayedMessageOutboxAcc = prevOutbox
	// Resolve message content from msgHash
	msgBytes, err := d.preimageResolver.ResolveTypedPreimage(arbutil.Keccak256PreimageType, msgHash)
	if err != nil {
		return nil, fmt.Errorf("error resolving delayed message content at index %d: %w", msgIndex, err)
	}
	delayed := new(mel.DelayedInboxMessage)
	if err = rlp.Decode(bytes.NewBuffer(msgBytes), delayed); err != nil {
		return nil, fmt.Errorf("failed to decode delayed message at index %d: %w", msgIndex, err)
	}
	return delayed, nil
}

// pourInboxToOutbox pours all items from the inbox into the outbox using preimage resolution.
func (d *DelayedMessageDatabase) pourInboxToOutbox(state *mel.State) error {
	inboxSize := state.DelayedMessagesSeen - state.DelayedMessagesRead
	if inboxSize == 0 {
		return nil
	}
	// Pop all items from inbox (LIFO: last-seen comes out first) and Push onto outbox
	// in original order (first-seen first → it ends up on top)
	curr := state.DelayedMessageInboxAcc
	for i := range inboxSize {
		result, err := d.preimageResolver.ResolveTypedPreimage(arbutil.Keccak256PreimageType, curr)
		if err != nil {
			return fmt.Errorf("error resolving inbox preimage during pour at position %d: %w", i, err)
		}
		prevAcc, msgHash, err := mel.SplitPreimage(result)
		if err != nil {
			return fmt.Errorf("inbox preimage at position %d: %w", i, err)
		}
		// Preimage is intentionally discarded: in replay/validation mode, all
		// outbox accumulator preimages were already recorded by native-mode
		// PourDelayedInboxToOutbox (state.go) during the recording step and are
		// available via the preimageResolver. We only recompute the hash here to
		// advance the accumulator; the resolver supplies preimages when
		// ReadDelayedMessage pops from the outbox.
		newAcc, _ := mel.HashChainLink(state.DelayedMessageOutboxAcc, msgHash)
		state.DelayedMessageOutboxAcc = newAcc
		curr = prevAcc
	}
	state.DelayedMessageInboxAcc = common.Hash{}
	return nil
}
