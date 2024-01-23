// Package boostpolicies defines a set of transaction scoring policies which take
// in a transaction type and output a single uint64 "score". This score can be used to order
// transactions by an ordering policy, such as timeboost, to be used in the Arbitrum sequencer.
package boostpolicies

import (
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
)

// BinaryExpressLaneScorer assigns a binary score to a tx of either 1 or 0. If a transaction
// is a tipping transaction type and has a non-zero gas tip cap, we give it a score of 1.
// Otherwise, it has a score of 0. This effectively creates an "express-lane" of transactions
// where users can buy a latency advantage in the sequence compared to other txs.
type BinaryExpressLaneScorer struct {
	AllowNonTippingTx bool
}

func (s *BinaryExpressLaneScorer) ScoreTx(tx *types.Transaction) uint64 {
	// As long as a tx is of the tipping tx type and has a non-zero bid,
	// it has a score of 1, otherwise it receives a score of 0.
	txTyp := tx.Type()
	okToScore := false
	isSubTyped := txTyp == types.ArbitrumSubtypedTxType
	if isSubTyped {
		subtype := types.GetArbitrumTxSubtype(tx)
		okToScore = subtype == types.ArbitrumTippingTxSubtype
	}
	if s.AllowNonTippingTx {
		okToScore = true
	}
	hasBid := tx.GasTipCap().Cmp(new(big.Int)) > 0
	if okToScore && hasBid {
		return 1
	}
	return 0
}

type BidScorer struct {
	AllowNonTippingTx bool
}

func (s *BidScorer) ScoreTx(tx *types.Transaction) uint64 {
	// As long as a tx is of the tipping tx type and has a non-zero bid,
	// it has a score of 1, otherwise it receives a score of 0.
	txTyp := tx.Type()
	okToScore := false
	isSubTyped := txTyp == types.ArbitrumSubtypedTxType
	if isSubTyped {
		subtype := types.GetArbitrumTxSubtype(tx)
		okToScore = subtype == types.ArbitrumTippingTxSubtype
	}
	if s.AllowNonTippingTx {
		okToScore = true
	}
	hasBid := tx.GasTipCap().Cmp(new(big.Int)) > 0
	if okToScore && hasBid {
		if !tx.GasTipCap().IsUint64() {
			return 0 // TODO: Handle this better.
		}
		return tx.GasTipCap().Uint64()
	}
	return 0
}

// NoopScorer assigns a single score of 0 to all transactions.
type NoopScorer struct{}

func (s *NoopScorer) ScoreTx(tx *types.Transaction) uint64 {
	return 0
}
