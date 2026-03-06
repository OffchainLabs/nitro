// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package main

import (
	"bytes"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbnode/mel"
	melextraction "github.com/offchainlabs/nitro/arbnode/mel/extraction"
	"github.com/offchainlabs/nitro/arbutil"
	melreplay "github.com/offchainlabs/nitro/mel-replay"
)

type delayedMessageDatabase struct {
	preimageResolver melreplay.PreimageResolver
}

func newDelayedMessageDatabase(preimageResolver melreplay.PreimageResolver) melextraction.DelayedMessageDatabase {
	return &delayedMessageDatabase{preimageResolver}
}

// ReadDelayedMessage pops the next delayed message from the outbox accumulator.
// If the outbox is empty, it pours the inbox into the outbox first using preimage resolution.
func (d *delayedMessageDatabase) ReadDelayedMessage(
	state *mel.State,
	msgIndex uint64,
) (*mel.DelayedInboxMessage, error) {
	if msgIndex >= state.DelayedMessagesSeen {
		return nil, fmt.Errorf("index %d out of range, total delayed messages seen: %d", msgIndex, state.DelayedMessagesSeen)
	}
	// Pour inbox to outbox if outbox is empty
	if state.DelayedMessageOutboxAcc == (common.Hash{}) && state.DelayedMessageInboxAcc != (common.Hash{}) {
		if err := d.pourInboxToOutbox(state); err != nil {
			return nil, fmt.Errorf("error pouring delayed inbox to outbox: %w", err)
		}
	}
	// Pop from outbox
	result, err := d.preimageResolver.ResolveTypedPreimage(arbutil.Keccak256PreimageType, state.DelayedMessageOutboxAcc)
	if err != nil {
		return nil, fmt.Errorf("error resolving outbox preimage for delayed message at index %d: %w", msgIndex, err)
	}
	if len(result) != 2*common.HashLength {
		return nil, fmt.Errorf("invalid outbox preimage length: %d, wanted %d", len(result), 2*common.HashLength)
	}
	prevOutbox := common.BytesToHash(result[:common.HashLength])
	msgHash := common.BytesToHash(result[common.HashLength:])
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
func (d *delayedMessageDatabase) pourInboxToOutbox(state *mel.State) error {
	inboxSize := state.DelayedMessagesSeen - state.DelayedMessagesRead
	if inboxSize == 0 {
		return nil
	}
	// Pop all items from inbox (LIFO: last-seen comes out first)
	msgHashes := make([]common.Hash, inboxSize)
	curr := state.DelayedMessageInboxAcc
	for i := inboxSize; i > 0; i-- {
		result, err := d.preimageResolver.ResolveTypedPreimage(arbutil.Keccak256PreimageType, curr)
		if err != nil {
			return fmt.Errorf("error resolving inbox preimage during pour at position %d: %w", i, err)
		}
		if len(result) != 2*common.HashLength {
			return fmt.Errorf("invalid inbox preimage length: %d, wanted %d", len(result), 2*common.HashLength)
		}
		prevAcc := common.BytesToHash(result[:common.HashLength])
		msgHash := common.BytesToHash(result[common.HashLength:])
		msgHashes[i-1] = msgHash
		curr = prevAcc
	}
	// Push items onto outbox in original order (first-seen first → it ends up on top)
	for _, msgHash := range msgHashes {
		preimage := append(state.DelayedMessageOutboxAcc.Bytes(), msgHash.Bytes()...)
		newAcc := crypto.Keccak256Hash(preimage)
		state.DelayedMessageOutboxAcc = newAcc
	}
	state.DelayedMessageInboxAcc = common.Hash{}
	return nil
}
