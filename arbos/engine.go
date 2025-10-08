// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbos

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/trie"
)

type Engine struct {
	IsSequencer bool
}

func (e Engine) Author(header *types.Header) (common.Address, error) {
	return header.Coinbase, nil
}

func (e Engine) VerifyHeader(chain consensus.ChainHeaderReader, header *types.Header) error {
	if header == nil {
		return errors.New("nil header")
	}

	// Enforce Arbitrum L2 invariants that don't require state.
	if header.UncleHash != types.EmptyUncleHash {
		return errors.New("uncles not supported")
	}
	if header.Difficulty == nil || header.Difficulty.Cmp(big.NewInt(1)) != 0 {
		return errors.New("unexpected difficulty")
	}
	if header.BaseFee == nil {
		return errors.New("missing basefee")
	}
	if header.GasLimit < header.GasUsed {
		return errors.New("gas used exceeds gas limit")
	}
	if header.Nonce != (types.BlockNonce{}) {
		return errors.New("unexpected nonce")
	}

	// Timestamp monotonicity relative to parent (if parent exists).
	if header.Number != nil && header.Number.Sign() > 0 {
		parentNumber := new(big.Int).Sub(header.Number, big.NewInt(1))
		parent := chain.GetHeader(header.ParentHash, parentNumber.Uint64())
		if parent != nil {
			if header.Time < parent.Time {
				return errors.New("timestamp older than parent")
			}
		}
	}

	return nil
}

func (e Engine) VerifyHeaders(chain consensus.ChainHeaderReader, headers []*types.Header) (chan<- struct{}, <-chan error) {
	abort := make(chan struct{})
	errs := make(chan error, len(headers))
	go func() {
		defer close(errs)
		for i := range headers {
			select {
			case <-abort:
				return
			default:
			}
			errs <- e.VerifyHeader(chain, headers[i])
		}
	}()
	return abort, errs
}

func (e Engine) VerifyUncles(chain consensus.ChainReader, block *types.Block) error {
	if len(block.Uncles()) != 0 {
		return errors.New("uncles not supported")
	}
	return nil
}

func (e Engine) Prepare(chain consensus.ChainHeaderReader, header *types.Header) error {
	header.Difficulty = big.NewInt(1)
	return nil
}

func (e Engine) Finalize(chain consensus.ChainHeaderReader, header *types.Header, state vm.StateDB, body *types.Body) {
	FinalizeBlock(header, body.Transactions, state, chain.Config())
}

func (e Engine) FinalizeAndAssemble(chain consensus.ChainHeaderReader, header *types.Header, state *state.StateDB, body *types.Body, receipts []*types.Receipt) (*types.Block, error) {

	e.Finalize(chain, header, state, body)

	block := types.NewBlock(header, &types.Body{Transactions: body.Transactions}, receipts, trie.NewStackTrie(nil))
	return block, nil
}

func (e Engine) Seal(chain consensus.ChainHeaderReader, block *types.Block, results chan<- *types.Block, stop <-chan struct{}) error {
	if !e.IsSequencer {
		return errors.New("sealing not supported")
	}
	if len(block.Transactions()) == 0 {
		return nil
	}
	results <- block
	return nil
}

func (e Engine) SealHash(header *types.Header) common.Hash {
	return header.Hash()
}

func (e Engine) CalcDifficulty(chain consensus.ChainHeaderReader, time uint64, parent *types.Header) *big.Int {
	return big.NewInt(1)
}

func (e Engine) APIs(chain consensus.ChainHeaderReader) []rpc.API {
	return nil
}

func (e Engine) Close() error {
	return nil
}
