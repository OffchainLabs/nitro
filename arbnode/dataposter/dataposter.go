// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package dataposter

import (
	"context"
	"flag"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/go-redis/redis/v8"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

type queuedTransaction struct {
	fullTx          *types.Transaction
	data            types.DynamicFeeTx
	endDataPosition uint64
	sent            bool
	created         time.Time // may be earlier than the tx was given to the tx poster
	nextReplacement time.Time
}

type DataPosterConfig struct {
	ReplacementInterval time.Duration `koanf:"replacement-interval"`
}

func DataPosterConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Duration(prefix+".replacement-interval", DefaultDataPosterConfig.ReplacementInterval, "transaction replace-by-fee interval")
}

var DefaultDataPosterConfig = DataPosterConfig{
	ReplacementInterval: time.Hour,
}

var TestDataPosterConfig = DataPosterConfig{
	ReplacementInterval: time.Second,
}

type DataPoster struct {
	stopwaiter.StopWaiter
	client arbutil.L1Interface
	auth   bind.TransactOpts
	config *DataPosterConfig
	redis  redis.UniversalClient // may be nil

	// these fields are protected by the mutex
	mutex     sync.Mutex
	lastBlock *big.Int
	balance   *big.Int
	nonce     uint64
	queue     []*queuedTransaction
}

func NewDataPoster(client arbutil.L1Interface, auth bind.TransactOpts, config *DataPosterConfig, redis redis.UniversalClient) *DataPoster {
	return &DataPoster{
		client: client,
		auth:   auth,
		config: config,
		redis:  redis,
	}
}

func (p *DataPoster) Initialize(ctx context.Context) error {
	nonce, err := p.client.NonceAt(ctx, p.auth.From, nil)
	if err != nil {
		return err
	}
	p.nonce = nonce
	if p.redis != nil {
		panic("TODO: query redis")
	}
	return nil
}

// Returns the end data position of the queue and true, or 0 and false if the queue is empty
func (p *DataPoster) GetNextNonceAndEndDataPosition(ctx context.Context, getEndDataPosition func(blockNum *big.Int) (uint64, error)) (uint64, uint64, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.updateState(ctx)
	if len(p.queue) > 0 {
		return p.nonce + uint64(len(p.queue)), p.queue[len(p.queue)-1].endDataPosition, nil
	}
	endDataPos, err := getEndDataPosition(p.lastBlock)
	return p.nonce, endDataPos, err
}

const extraGas uint64 = 50_000

func (p *DataPoster) PostTransaction(ctx context.Context, dataCreatedAt time.Time, endDataPosition uint64, to common.Address, calldata []byte, gasLimit uint64) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	inner := types.DynamicFeeTx{
		Nonce:     p.nonce + uint64(len(p.queue)),
		GasTipCap: TODO,
		GasFeeCap: TODO,
		Gas:       gasLimit,
		To:        &to,
		Value:     new(big.Int),
		Data:      calldata,
	}
	fullTx, err := p.auth.Signer(p.auth.From, types.NewTx(&inner))
	if err != nil {
		return err
	}
	queuedTx := queuedTransaction{
		data:            inner,
		fullTx:          fullTx,
		endDataPosition: endDataPosition,
		sent:            false,
		created:         dataCreatedAt,
		nextReplacement: time.Now().Add(p.config.ReplacementInterval),
	}
	return p.sendTx(ctx, len(p.queue), &queuedTx)
}

// the mutex must be held by the caller
func (p *DataPoster) sendTx(ctx context.Context, idx int, newTx *queuedTransaction) error {
	if p.redis != nil {
		panic("TODO: store tx in redis")
	}
	if idx == len(p.queue) {
		p.queue = append(p.queue, newTx)
	} else {
		p.queue[idx] = newTx
	}
	err := p.client.SendTransaction(ctx, newTx.fullTx)
	newTx.sent = err == nil
	return err
}

const minRbfIncrease arbmath.Bips = arbmath.OneInBips * 11 / 10

// the mutex must be held by the caller
func (p *DataPoster) replaceTx(ctx context.Context, idx int) error {
	latestHeader, err := p.client.HeaderByNumber(ctx, nil)
	if err != nil {
		return err
	}
	tx := p.queue[idx]
	newTipCap := new(big.Int).Mul(tx.data.GasTipCap, big.NewInt(11))
	newTipCap.Div(newTipCap, big.NewInt(10))
	newFeeCap := new(big.Int).Mul(latestHeader.BaseFee, big.NewInt(2))
	newFeeCap.Add(newFeeCap, newTipCap)

	desiredFeeCap := newFeeCap
	maxFeeCap := new(big.Int).Div(p.balance, new(big.Int).SetUint64(tx.data.Gas))
	newFeeCap = arbmath.BigMin(newFeeCap, maxFeeCap)
	minNewFeeCap := arbmath.BigMulByBips(tx.data.GasFeeCap, minRbfIncrease)
	if newFeeCap.Cmp(minNewFeeCap) < 0 {
		if desiredFeeCap.Cmp(minNewFeeCap) >= 0 {
			log.Error(
				"lack of L1 balance prevents posting transaction with a higher fee cap",
				"balance", p.balance,
				"gasLimit", tx.data.Gas,
				"desiredFeeCap", desiredFeeCap,
				"maxFeeCap", maxFeeCap,
			)
		}
		tx.nextReplacement = time.Now().Add(time.Minute)
		return nil
	}

	newTx := *tx
	newTx.nextReplacement = time.Now().Add(p.config.ReplacementInterval)
	newTx.sent = false
	newTx.data.GasFeeCap = newFeeCap
	newTx.data.GasTipCap = newTipCap
	newTx.fullTx, err = p.auth.Signer(p.auth.From, types.NewTx(&newTx.data))
	if err != nil {
		return err
	}

	return p.sendTx(ctx, idx, &newTx)
}

var l1BlockLookBehind = big.NewInt(2)

// the mutex must be held by the caller
func (p *DataPoster) updateState(ctx context.Context) error {
	header, err := p.client.HeaderByNumber(ctx, nil)
	if err != nil {
		return err
	}
	p.lastBlock = arbmath.BigSub(header.Number, l1BlockLookBehind)
	nonce, err := p.client.NonceAt(ctx, p.auth.From, p.lastBlock)
	if err != nil {
		return err
	}
	if nonce > p.nonce {
		confirmed := int(nonce - p.nonce)
		if len(p.queue) > confirmed {
			p.queue = p.queue[confirmed:]
		} else {
			p.queue = p.queue[:0]
		}
		p.nonce = nonce
	}
	balance, err := p.client.BalanceAt(ctx, p.auth.From, p.lastBlock)
	if err != nil {
		return err
	}
	p.balance = balance
	return nil
}

const minWait = time.Second * 10

func (p *DataPoster) Start(ctxIn context.Context) {
	p.StopWaiter.Start(ctxIn)
	p.CallIteratively(func(ctx context.Context) time.Duration {
		p.mutex.Lock()
		defer p.mutex.Unlock()
		err := p.updateState(ctx)
		if err != nil {
			log.Warn("failed to update tx poster internal state", "err", err)
			return minWait
		}
		now := time.Now()
		nextCheck := now.Add(time.Hour)
		for i, tx := range p.queue {
			if now.After(tx.nextReplacement) {
				err := p.replaceTx(ctx, i)
				if err != nil {
					log.Error("failed to replace-by-fee transaction", "err", err)
				}
			}
			if nextCheck.After(tx.nextReplacement) {
				nextCheck = tx.nextReplacement
			}
			if !tx.sent {
				err := p.client.SendTransaction(ctx, tx.fullTx)
				if err != nil {
					log.Warn("failed to re-send transaction", "err", err)
					nextSend := time.Now().Add(time.Minute)
					if nextCheck.After(nextSend) {
						nextCheck = nextSend
					}
				} else {
					tx.sent = true
				}
			}
		}
		wait := nextCheck.Sub(time.Now())
		if wait < minWait {
			wait = minWait
		}
		return wait
	})
}
