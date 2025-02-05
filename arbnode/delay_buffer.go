// Copyright 2024, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

// This file contains functions related to the delay buffer feature that are used mostly in the
// batch poster.

package arbnode

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/offchainlabs/bold/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/util/headerreader"
)

// DelayBufferConfig originates from the sequencer inbox contract.
type DelayBufferConfig struct {
	Enabled   bool
	Threshold uint64
}

// GetDelayBufferConfig gets the delay buffer config from the sequencer inbox contract.
// If the contract doesn't support the delay buffer, it returns a config with Enabled set to false.
func GetDelayBufferConfig(ctx context.Context, sequencerInbox *bridgegen.SequencerInbox) (
	*DelayBufferConfig, error) {

	callOpts := bind.CallOpts{Context: ctx}
	enabled, err := sequencerInbox.IsDelayBufferable(&callOpts)
	if err != nil {
		if headerreader.ExecutionRevertedRegexp.MatchString(err.Error()) {
			return &DelayBufferConfig{Enabled: false}, nil
		}
		return nil, fmt.Errorf("retrieve SequencerInbox.isDelayBufferable: %w", err)
	}
	if !enabled {
		return &DelayBufferConfig{Enabled: false}, nil
	}
	bufferData, err := sequencerInbox.Buffer(&callOpts)
	if err != nil {
		return nil, fmt.Errorf("retrieve SequencerInbox.buffer: %w", err)
	}
	config := &DelayBufferConfig{
		Enabled:   true,
		Threshold: bufferData.Threshold,
	}
	return config, nil
}

// GenDelayProof generates the delay proof based on batch's first delayed message and the delayed
// accumulater from the inbox.
func GenDelayProof(ctx context.Context, message *arbostypes.MessageWithMetadata, inbox *InboxTracker) (
	*bridgegen.DelayProof, error) {

	if message.DelayedMessagesRead == 0 {
		return nil, fmt.Errorf("BUG: trying to generate delay proof without delayed message")
	}
	seqNum := message.DelayedMessagesRead - 1
	var beforeDelayedAcc common.Hash
	if seqNum > 0 {
		var err error
		beforeDelayedAcc, err = inbox.GetDelayedAcc(seqNum - 1)
		if err != nil {
			return nil, err
		}
	}
	delayedMessage := bridgegen.MessagesMessage{
		Kind:            message.Message.Header.Kind,
		Sender:          message.Message.Header.Poster,
		BlockNumber:     message.Message.Header.BlockNumber,
		Timestamp:       message.Message.Header.Timestamp,
		InboxSeqNum:     new(big.Int).SetUint64(seqNum),
		BaseFeeL1:       message.Message.Header.L1BaseFee,
		MessageDataHash: crypto.Keccak256Hash(message.Message.L2msg),
	}
	delayProof := &bridgegen.DelayProof{
		BeforeDelayedAcc: beforeDelayedAcc,
		DelayedMessage:   delayedMessage,
	}
	return delayProof, nil
}
