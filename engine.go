package arbstate

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/offchainlabs/arbstate/arbos2"
)

type Engine struct{}

func (e Engine) Author(header *types.Header) (common.Address, error) {
	return header.Coinbase, nil
}

func (e Engine) VerifyHeader(chain consensus.ChainHeaderReader, header *types.Header, seal bool) error {
	// TODO what verification should be done here?
	return nil
}

func (e Engine) VerifyHeaders(chain consensus.ChainHeaderReader, headers []*types.Header, seals []bool) (chan<- struct{}, <-chan error) {
	errors := make(chan error, len(headers))
	for i := range headers {
		errors <- e.VerifyHeader(chain, headers[i], seals[i])
	}
	return make(chan struct{}), errors
}

func (e Engine) VerifyUncles(chain consensus.ChainReader, block *types.Block) error {
	if len(block.Uncles()) != 0 {
		return errors.New("uncles not supported")
	}
	return nil
}

func (e Engine) Prepare(chain consensus.ChainHeaderReader, header *types.Header) error {
	return nil
}

func (e Engine) Finalize(chain consensus.ChainHeaderReader, header *types.Header, state *state.StateDB, txs []*types.Transaction, uncles []*types.Header, receipts []*types.Receipt) {
	arbos2.Finalize(header, state, txs, receipts)
	header.Root = state.IntermediateRoot(true)
}

func (e Engine) FinalizeAndAssemble(chain consensus.ChainHeaderReader, header *types.Header, state *state.StateDB, txs []*types.Transaction,
	uncles []*types.Header, receipts []*types.Receipt) (*types.Block, error) {

	e.Finalize(chain, header, state, txs, uncles, receipts)

	block := types.NewBlock(header, txs, nil, receipts, trie.NewStackTrie(nil))
	return block, nil
}

func (e Engine) Seal(chain consensus.ChainHeaderReader, block *types.Block, results chan<- *types.Block, stop <-chan struct{}) error {
	return errors.New("sealing not supported")
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
