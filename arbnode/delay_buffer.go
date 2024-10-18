// Copyright 2024, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

// This file contains functions related to the delay buffer feature that are used mostly in the
// batch poster.

package arbnode

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/util/headerreader"
)

// DelayBufferConfig originates from the sequencer inbox contract.
type DelayBufferConfig struct {
	Enabled   bool
	Threshold uint64
}

// GetBufferConfig gets the delay buffer config from the sequencer inbox contract.
// If the contract doesn't support the delay buffer, it returns a config with Enabled set to false.
func GetDelayBufferConfig(ctx context.Context, client *ethclient.Client, sequencerInboxAddress common.Address) (
	*DelayBufferConfig, error) {

	sequencerInbox, err := bridgegen.NewSequencerInbox(sequencerInboxAddress, client)
	if err != nil {
		return nil, fmt.Errorf("create sequencer inbox binding: %w", err)
	}
	callOpts := bind.CallOpts{
		Context: ctx,
	}
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
