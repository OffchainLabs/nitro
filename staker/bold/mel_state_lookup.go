// Copyright 2023-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package bold

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/offchainlabs/nitro/bold/state"
	"github.com/offchainlabs/nitro/staker"
)

// Compile-time check that melStateLookup implements ValidatedMELStateLookup.
var _ state.ValidatedMELStateLookup = (*melStateLookup)(nil)

// melStateLookup implements state.ValidatedMELStateLookup by resolving parent
// chain block hashes to block numbers and delegating to the MEL validator for
// validated state retrieval.
type melStateLookup struct {
	melValidator   staker.MELValidatorInterface
	parentChainRPC *ethclient.Client
}

// NewMELStateLookup creates a ValidatedMELStateLookup backed by the given MEL
// validator and parent chain client.
func NewMELStateLookup(
	melValidator staker.MELValidatorInterface,
	parentChainRPC *ethclient.Client,
) state.ValidatedMELStateLookup {
	return &melStateLookup{
		melValidator:   melValidator,
		parentChainRPC: parentChainRPC,
	}
}

// GetValidatedMELStateByBlockHash resolves a parent chain block hash to a block
// number, then retrieves the MEL state at that block — but only if MEL
// validation has reached it. Returns state.ErrChainCatchingUp if not.
func (m *melStateLookup) GetValidatedMELStateByBlockHash(ctx context.Context, blockHash common.Hash) (*state.MELStateInfo, error) {
	if blockHash == (common.Hash{}) {
		return nil, fmt.Errorf("cannot look up MEL state for zero block hash")
	}
	header, err := m.parentChainRPC.HeaderByHash(ctx, blockHash)
	if err != nil {
		return nil, fmt.Errorf("could not resolve parent chain block hash %s: %w", blockHash, err)
	}
	if !header.Number.IsInt64() {
		return nil, fmt.Errorf("parent chain block number not representable: %s", header.Number)
	}
	blockNum := new(big.Int).Set(header.Number).Uint64()

	melState, err := m.melValidator.GetValidatedMELStateAtBlock(ctx, blockNum)
	if err != nil {
		return nil, err
	}
	return &state.MELStateInfo{
		BatchCount:             melState.BatchCount,
		MsgCount:               melState.MsgCount,
		ParentChainBlockNumber: melState.ParentChainBlockNumber,
		StateHash:              melState.Hash(),
	}, nil
}
