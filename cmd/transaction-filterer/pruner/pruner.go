// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

// Package pruner walks the parent chain delayed inbox in order and submits
// DeleteFilteredTransaction for any transaction still marked as filtered in
// ArbFilteredTransactionsManager. A delayed message is only processed once
// it has been sequenced on the child chain, observed via the
// DelayedMessagesRead counter stored in the finalized L2 header nonce.
package pruner

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbnode/mel"
	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

var finalizedTag = big.NewInt(rpc.FinalizedBlockNumber.Int64())

type headerReader interface {
	HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error)
}

type delayedBridge interface {
	LookupMessagesInRange(ctx context.Context, from, to *big.Int, bf arbostypes.FallibleBatchFetcherWithParentBlock) ([]*mel.DelayedInboxMessage, error)
}

type filterManager interface {
	IsTransactionFiltered(opts *bind.CallOpts, txHash [32]byte) (bool, error)
	DeleteFilteredTransaction(opts *bind.TransactOpts, txHash [32]byte) (*types.Transaction, error)
}

type Pruner struct {
	stopwaiter.StopWaiter

	config  *Config
	chainID *big.Int
	parent  headerReader
	child   headerReader
	bridge  delayedBridge
	manager filterManager
	txOpts  *bind.TransactOpts

	nextIdx   uint64
	scanBlock uint64
}

func New(config *Config, parent, child *ethclient.Client, txOpts *bind.TransactOpts, chainID *big.Int) (*Pruner, error) {
	bridge, err := arbnode.NewDelayedBridge(parent, common.HexToAddress(config.BridgeAddress), 0)
	if err != nil {
		return nil, fmt.Errorf("creating delayed bridge: %w", err)
	}
	manager, err := precompilesgen.NewArbFilteredTransactionsManager(types.ArbFilteredTransactionsManagerAddress, child)
	if err != nil {
		return nil, fmt.Errorf("creating filter manager: %w", err)
	}
	return &Pruner{
		config:  config,
		chainID: chainID,
		parent:  parent,
		child:   child,
		bridge:  bridge,
		manager: manager,
		txOpts:  txOpts,
		nextIdx: config.StartDelayedMessageIndex,
	}, nil
}

func (p *Pruner) Start(ctx context.Context) {
	p.StopWaiter.Start(ctx, p)
	p.CallIteratively(p.step)
}

func (p *Pruner) step(ctx context.Context) time.Duration {
	childHead, err := p.child.HeaderByNumber(ctx, finalizedTag)
	if err != nil {
		log.Error("pruner: child finalized header", "err", err)
		return p.config.PollInterval
	}
	delayedMessagesRead := childHead.Nonce.Uint64()
	if p.nextIdx >= delayedMessagesRead {
		return p.config.PollInterval
	}

	parentHead, err := p.parent.HeaderByNumber(ctx, finalizedTag)
	if err != nil {
		log.Error("pruner: parent finalized header", "err", err)
		return p.config.PollInterval
	}
	parentFinalized := parentHead.Number.Uint64()
	if p.scanBlock > parentFinalized {
		return p.config.PollInterval
	}

	from := p.scanBlock
	to := min(from+p.config.ParentChainScanRange, parentFinalized)
	msgs, err := p.bridge.LookupMessagesInRange(ctx, new(big.Int).SetUint64(from), new(big.Int).SetUint64(to), nil)
	if err != nil {
		log.Error("pruner: lookup delayed messages", "from", from, "to", to, "err", err)
		return p.config.PollInterval
	}

	for _, msg := range msgs {
		idx := msg.Message.Header.RequestId.Big().Uint64()
		if idx < p.nextIdx {
			continue
		}
		if idx != p.nextIdx {
			log.Error("pruner: gap in delayed messages", "want", p.nextIdx, "got", idx)
			return p.config.PollInterval
		}
		if idx >= delayedMessagesRead {
			// Known to L1 but not yet sequenced on L2; re-scan this parent block next iteration.
			p.scanBlock = msg.ParentChainBlockNumber
			return p.config.PollInterval
		}
		if err := p.processMessage(ctx, msg, childHead.Number); err != nil {
			log.Error("pruner: process delayed message", "idx", idx, "err", err)
			return p.config.PollInterval
		}
		p.nextIdx++
	}

	p.scanBlock = to + 1
	if p.nextIdx < delayedMessagesRead && p.scanBlock <= parentFinalized {
		return 0
	}
	return p.config.PollInterval
}

// processMessage parses msg and submits DeleteFilteredTransaction for each
// transaction still flagged as filtered at childFinalized. lastArbosVersion=0
// is safe: it only affects L1MessageType_BatchPostingReport, an internal
// message type whose transaction is never subject to filtering.
func (p *Pruner) processMessage(ctx context.Context, msg *mel.DelayedInboxMessage, childFinalized *big.Int) error {
	txs, err := arbos.ParseL2Transactions(msg.Message, p.chainID, 0)
	if err != nil {
		return fmt.Errorf("parsing delayed message: %w", err)
	}
	callOpts := &bind.CallOpts{Context: ctx, BlockNumber: childFinalized}
	for _, tx := range txs {
		hash := tx.Hash()
		filtered, err := p.manager.IsTransactionFiltered(callOpts, hash)
		if err != nil {
			return fmt.Errorf("IsTransactionFiltered %v: %w", hash.Hex(), err)
		}
		if !filtered {
			continue
		}
		txOpts := *p.txOpts
		txOpts.Context = ctx
		sent, err := p.manager.DeleteFilteredTransaction(&txOpts, hash)
		if err != nil {
			return fmt.Errorf("DeleteFilteredTransaction %v: %w", hash.Hex(), err)
		}
		log.Info("pruner: submitted delete filtered transaction", "txHashDeleted", hash.Hex(), "submittedTxHash", sent.Hash().Hex())
	}
	return nil
}
