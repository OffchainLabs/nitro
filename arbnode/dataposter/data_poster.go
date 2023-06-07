// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package dataposter

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/go-redis/redis/v8"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/signature"
	"github.com/offchainlabs/nitro/util/stopwaiter"
	flag "github.com/spf13/pflag"
)

type queuedTransaction[Meta any] struct {
	FullTx          *types.Transaction
	Data            types.DynamicFeeTx
	Meta            Meta
	Sent            bool
	Created         time.Time // may be earlier than the tx was given to the tx poster
	NextReplacement time.Time
}

type QueueStorage[Item any] interface {
	GetContents(ctx context.Context, startingIndex uint64, maxResults uint64) ([]*Item, error)
	GetLast(ctx context.Context) (*Item, error)
	Prune(ctx context.Context, keepStartingAt uint64) error
	Put(ctx context.Context, index uint64, prevItem *Item, newItem *Item) error
	Length(ctx context.Context) (int, error)
	IsPersistent() bool
}

type DataPosterConfig struct {
	RedisSigner            signature.SimpleHmacConfig `koanf:"redis-signer"`
	ReplacementTimes       string                     `koanf:"replacement-times"`
	WaitForL1Finality      bool                       `koanf:"wait-for-l1-finality" reload:"hot"`
	MaxMempoolTransactions uint64                     `koanf:"max-mempool-transactions" reload:"hot"`
	MaxQueuedTransactions  int                        `koanf:"max-queued-transactions" reload:"hot"`
	TargetPriceGwei        float64                    `koanf:"target-price-gwei" reload:"hot"`
	UrgencyGwei            float64                    `koanf:"urgency-gwei" reload:"hot"`
	MinFeeCapGwei          float64                    `koanf:"min-fee-cap-gwei" reload:"hot"`
	MinTipCapGwei          float64                    `koanf:"min-tip-cap-gwei" reload:"hot"`
}

type DataPosterConfigFetcher func() *DataPosterConfig

func DataPosterConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.String(prefix+".replacement-times", DefaultDataPosterConfig.ReplacementTimes, "comma-separated list of durations since first posting to attempt a replace-by-fee")
	f.Bool(prefix+".wait-for-l1-finality", DefaultDataPosterConfig.WaitForL1Finality, "only treat a transaction as confirmed after L1 finality has been achieved (recommended)")
	f.Uint64(prefix+".max-mempool-transactions", DefaultDataPosterConfig.MaxMempoolTransactions, "the maximum number of transactions to have queued in the mempool at once (0 = unlimited)")
	f.Int(prefix+".max-queued-transactions", DefaultDataPosterConfig.MaxQueuedTransactions, "the maximum number of unconfirmed transactions to track at once (0 = unlimited)")
	f.Float64(prefix+".target-price-gwei", DefaultDataPosterConfig.TargetPriceGwei, "the target price to use for maximum fee cap calculation")
	f.Float64(prefix+".urgency-gwei", DefaultDataPosterConfig.UrgencyGwei, "the urgency to use for maximum fee cap calculation")
	f.Float64(prefix+".min-fee-cap-gwei", DefaultDataPosterConfig.MinFeeCapGwei, "the minimum fee cap to post transactions at")
	f.Float64(prefix+".min-tip-cap-gwei", DefaultDataPosterConfig.MinTipCapGwei, "the minimum tip cap to post transactions at")
	signature.SimpleHmacConfigAddOptions(prefix+".redis-signer", f)
}

var DefaultDataPosterConfig = DataPosterConfig{
	ReplacementTimes:       "5m,10m,20m,30m,1h,2h,4h,6h,8h,12h,16h,18h,20h,22h",
	WaitForL1Finality:      true,
	TargetPriceGwei:        60.,
	UrgencyGwei:            2.,
	MaxMempoolTransactions: 64,
	MinTipCapGwei:          0.05,
}

var TestDataPosterConfig = DataPosterConfig{
	ReplacementTimes:       "1s,2s,5s,10s,20s,30s,1m,5m",
	RedisSigner:            signature.TestSimpleHmacConfig,
	WaitForL1Finality:      false,
	TargetPriceGwei:        60.,
	UrgencyGwei:            2.,
	MaxMempoolTransactions: 64,
	MinTipCapGwei:          0.05,
}

// DataPoster must be RLP serializable and deserializable
type DataPoster[Meta any] struct {
	stopwaiter.StopWaiter
	headerReader      *headerreader.HeaderReader
	client            arbutil.L1Interface
	auth              *bind.TransactOpts
	redisLock         AttemptLocker
	config            DataPosterConfigFetcher
	replacementTimes  []time.Duration
	metadataRetriever func(ctx context.Context, blockNum *big.Int) (Meta, error)

	// these fields are protected by the mutex
	mutex      sync.Mutex
	lastBlock  *big.Int
	balance    *big.Int
	nonce      uint64
	queue      QueueStorage[queuedTransaction[Meta]]
	errorCount map[uint64]int // number of consecutive intermittent errors rbf-ing or sending, per nonce
}

type AttemptLocker interface {
	AttemptLock(context.Context) bool
}

func NewDataPoster[Meta any](headerReader *headerreader.HeaderReader, auth *bind.TransactOpts, redisClient redis.UniversalClient, redisLock AttemptLocker, config DataPosterConfigFetcher, metadataRetriever func(ctx context.Context, blockNum *big.Int) (Meta, error)) (*DataPoster[Meta], error) {
	var replacementTimes []time.Duration
	var lastReplacementTime time.Duration
	for _, s := range strings.Split(config().ReplacementTimes, ",") {
		t, err := time.ParseDuration(s)
		if err != nil {
			return nil, err
		}
		if t <= lastReplacementTime {
			return nil, errors.New("replacement times must be increasing")
		}
		replacementTimes = append(replacementTimes, t)
		lastReplacementTime = t
	}
	if len(replacementTimes) == 0 {
		log.Warn("disabling replace-by-fee for data poster")
	}
	// To avoid special casing "don't replace again", replace in 10 years
	replacementTimes = append(replacementTimes, time.Hour*24*365*10)
	var queue QueueStorage[queuedTransaction[Meta]]
	if redisClient == nil {
		queue = NewSliceStorage[queuedTransaction[Meta]]()
	} else {
		var err error
		queue, err = NewRedisStorage[queuedTransaction[Meta]](redisClient, "data-poster.queue", &config().RedisSigner)
		if err != nil {
			return nil, err
		}
	}
	return &DataPoster[Meta]{
		headerReader:      headerReader,
		client:            headerReader.Client(),
		auth:              auth,
		config:            config,
		replacementTimes:  replacementTimes,
		metadataRetriever: metadataRetriever,
		queue:             queue,
		redisLock:         redisLock,
		errorCount:        make(map[uint64]int),
	}, nil
}

func (p *DataPoster[Meta]) From() common.Address {
	return p.auth.From
}

func (p *DataPoster[Meta]) GetNextNonceAndMeta(ctx context.Context) (uint64, Meta, error) {
	config := p.config()
	var emptyMeta Meta
	p.mutex.Lock()
	defer p.mutex.Unlock()
	blockNum, err := p.client.BlockNumber(ctx)
	if err != nil {
		return 0, emptyMeta, err
	}
	lastQueueItem, err := p.queue.GetLast(ctx)
	if err != nil {
		return 0, emptyMeta, err
	}
	if lastQueueItem != nil {
		nextNonce := lastQueueItem.Data.Nonce + 1
		if config.MaxQueuedTransactions > 0 {
			queueLen, err := p.queue.Length(ctx)
			if err != nil {
				return 0, emptyMeta, err
			}
			if queueLen >= config.MaxQueuedTransactions {
				return 0, emptyMeta, fmt.Errorf("attempting to post a transaction with nonce %v while current nonce is %v would exceed max data poster queue length of %v", nextNonce, p.nonce, config.MaxQueuedTransactions)
			}
		}
		if config.MaxMempoolTransactions > 0 {
			unconfirmedNonce, err := p.client.NonceAt(ctx, p.auth.From, nil)
			if err != nil {
				return 0, emptyMeta, fmt.Errorf("failed to get unconfirmed nonce: %w", err)
			}
			if nextNonce >= unconfirmedNonce+config.MaxMempoolTransactions {
				return 0, emptyMeta, fmt.Errorf("attempting to post a transaction with nonce %v while unconfirmed nonce is %v would exceed max mempool transactions of %v", nextNonce, unconfirmedNonce, config.MaxMempoolTransactions)
			}
		}
		return nextNonce, lastQueueItem.Meta, nil
	}
	err = p.updateNonce(ctx)
	if err != nil {
		if !p.queue.IsPersistent() && config.WaitForL1Finality {
			return 0, emptyMeta, fmt.Errorf("error getting latest finalized nonce (and queue is not persistent): %w", err)
		}
		// Fall back to using a recent block to get the nonce. This is safe because there's nothing in the queue.
		nonceQueryBlock := arbmath.UintToBig(arbmath.SaturatingUSub(blockNum, 1))
		log.Warn("failed to update nonce with queue empty; falling back to using a recent block", "recentBlock", nonceQueryBlock, "err", err)
		nonce, err := p.client.NonceAt(ctx, p.auth.From, nonceQueryBlock)
		if err != nil {
			return 0, emptyMeta, fmt.Errorf("failed to get nonce at block %v: %w", nonceQueryBlock, err)
		}
		p.lastBlock = nonceQueryBlock
		p.nonce = nonce
	}
	meta, err := p.metadataRetriever(ctx, p.lastBlock)
	return p.nonce, meta, err
}

const minRbfIncrease = arbmath.OneInBips * 11 / 10

func (p *DataPoster[Meta]) getFeeAndTipCaps(ctx context.Context, gasLimit uint64, lastFeeCap *big.Int, lastTipCap *big.Int, dataCreatedAt time.Time, backlogOfBatches uint64) (*big.Int, *big.Int, error) {
	config := p.config()
	latestHeader, err := p.headerReader.LastHeader(ctx)
	if err != nil {
		return nil, nil, err
	}
	newFeeCap := new(big.Int).Mul(latestHeader.BaseFee, big.NewInt(2))
	newFeeCap = arbmath.BigMax(newFeeCap, arbmath.FloatToBig(config.MinFeeCapGwei*params.GWei))

	newTipCap, err := p.client.SuggestGasTipCap(ctx)
	if err != nil {
		return nil, nil, err
	}
	newTipCap = arbmath.BigMax(newTipCap, arbmath.FloatToBig(config.MinTipCapGwei*params.GWei))

	hugeTipIncrease := false
	if lastTipCap != nil {
		newTipCap = arbmath.BigMax(newTipCap, arbmath.BigMulByBips(lastTipCap, minRbfIncrease))
		// hugeTipIncrease is true if the new tip cap is at least 10x the last tip cap
		hugeTipIncrease = lastTipCap.Sign() == 0 || arbmath.BigDiv(newTipCap, lastTipCap).Cmp(big.NewInt(10)) >= 0
	}

	newFeeCap.Add(newFeeCap, newTipCap)
	if lastFeeCap != nil && hugeTipIncrease {
		log.Warn("data poster recommending huge tip increase", "lastTipCap", lastTipCap, "newTipCap", newTipCap)
		// If we're trying to drastically increase the tip, make sure we increase the fee cap by minRbfIncrease.
		newFeeCap = arbmath.BigMax(newFeeCap, arbmath.BigMulByBips(lastFeeCap, minRbfIncrease))
	}

	elapsed := time.Since(dataCreatedAt)
	// MaxFeeCap = (BacklogOfBatches^2 * UrgencyGWei^2 + TargetPriceGWei) * GWei
	maxFeeCap :=
		arbmath.FloatToBig(
			(float64(arbmath.SquareUint(backlogOfBatches))*
				arbmath.SquareFloat(config.UrgencyGwei) +
				config.TargetPriceGwei) *
				params.GWei)
	if arbmath.BigGreaterThan(newFeeCap, maxFeeCap) {
		log.Warn(
			"reducing proposed fee cap to current maximum",
			"proposedFeeCap", newFeeCap,
			"maxFeeCap", maxFeeCap,
			"elapsed", elapsed,
		)
		newFeeCap = maxFeeCap
	}

	balanceFeeCap := new(big.Int).Div(p.balance, new(big.Int).SetUint64(gasLimit))
	if arbmath.BigGreaterThan(newFeeCap, balanceFeeCap) {
		log.Error(
			"lack of L1 balance prevents posting transaction with desired fee cap",
			"balance", p.balance,
			"gasLimit", gasLimit,
			"desiredFeeCap", newFeeCap,
			"balanceFeeCap", balanceFeeCap,
		)
		newFeeCap = balanceFeeCap
	}

	if arbmath.BigGreaterThan(newTipCap, newFeeCap) {
		log.Warn(
			"reducing new tip cap to new fee cap",
			"proposedTipCap", newTipCap,
			"newFeeCap", newFeeCap,
		)
		newTipCap = new(big.Int).Set(newFeeCap)
	}

	return newFeeCap, newTipCap, nil
}

func (p *DataPoster[Meta]) PostTransaction(ctx context.Context, dataCreatedAt time.Time, nonce uint64, meta Meta, to common.Address, calldata []byte, gasLimit uint64) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	err := p.updateBalance(ctx)
	if err != nil {
		return fmt.Errorf("failed to update data poster balance: %w", err)
	}
	feeCap, tipCap, err := p.getFeeAndTipCaps(ctx, gasLimit, nil, nil, dataCreatedAt, 0)
	if err != nil {
		return err
	}
	inner := types.DynamicFeeTx{
		Nonce:     nonce,
		GasTipCap: tipCap,
		GasFeeCap: feeCap,
		Gas:       gasLimit,
		To:        &to,
		Value:     new(big.Int),
		Data:      calldata,
	}
	fullTx, err := p.auth.Signer(p.auth.From, types.NewTx(&inner))
	if err != nil {
		return err
	}
	queuedTx := queuedTransaction[Meta]{
		Data:            inner,
		FullTx:          fullTx,
		Meta:            meta,
		Sent:            false,
		Created:         dataCreatedAt,
		NextReplacement: time.Now().Add(p.replacementTimes[0]),
	}
	return p.sendTx(ctx, nil, &queuedTx)
}

// the mutex must be held by the caller
func (p *DataPoster[Meta]) saveTx(ctx context.Context, prevTx *queuedTransaction[Meta], newTx *queuedTransaction[Meta]) error {
	if prevTx != nil && prevTx.Data.Nonce != newTx.Data.Nonce {
		return fmt.Errorf("prevTx nonce %v doesn't match newTx nonce %v", prevTx.Data.Nonce, newTx.Data.Nonce)
	}
	return p.queue.Put(ctx, newTx.Data.Nonce, prevTx, newTx)
}

func (p *DataPoster[Meta]) sendTx(ctx context.Context, prevTx *queuedTransaction[Meta], newTx *queuedTransaction[Meta]) error {
	if prevTx != newTx {
		err := p.saveTx(ctx, prevTx, newTx)
		if err != nil {
			return err
		}
	}
	err := p.client.SendTransaction(ctx, newTx.FullTx)
	if err != nil {
		if strings.Contains(err.Error(), "already known") || strings.Contains(err.Error(), "nonce too low") {
			log.Info("DataPoster transaction already known", "err", err, "nonce", newTx.FullTx.Nonce(), "hash", newTx.FullTx.Hash())
			err = nil
		} else {
			log.Warn("DataPoster failed to send transaction", "err", err, "nonce", newTx.FullTx.Nonce(), "feeCap", newTx.FullTx.GasFeeCap(), "tipCap", newTx.FullTx.GasTipCap())
			return err
		}
	} else {
		log.Info("DataPoster sent transaction", "nonce", newTx.FullTx.Nonce(), "hash", newTx.FullTx.Hash(), "feeCap", newTx.FullTx.GasFeeCap())
	}
	newerTx := *newTx
	newerTx.Sent = true
	return p.saveTx(ctx, newTx, &newerTx)
}

// the mutex must be held by the caller
func (p *DataPoster[Meta]) replaceTx(ctx context.Context, prevTx *queuedTransaction[Meta], backlogOfBatches uint64) error {
	newFeeCap, newTipCap, err := p.getFeeAndTipCaps(ctx, prevTx.Data.Gas, prevTx.Data.GasFeeCap, prevTx.Data.GasTipCap, prevTx.Created, backlogOfBatches)
	if err != nil {
		return err
	}

	minNewFeeCap := arbmath.BigMulByBips(prevTx.Data.GasFeeCap, minRbfIncrease)
	newTx := *prevTx
	if newFeeCap.Cmp(minNewFeeCap) < 0 {
		log.Debug(
			"no need to replace by fee transaction",
			"nonce", prevTx.Data.Nonce,
			"lastFeeCap", prevTx.Data.GasFeeCap,
			"recommendedFeeCap", newFeeCap,
			"lastTipCap", prevTx.Data.GasTipCap,
			"recommendedTipCap", newTipCap,
		)
		newTx.NextReplacement = time.Now().Add(time.Minute)
		return p.sendTx(ctx, prevTx, &newTx)
	}

	elapsed := time.Since(prevTx.Created)
	for _, replacement := range p.replacementTimes {
		if elapsed >= replacement {
			continue
		}
		newTx.NextReplacement = prevTx.Created.Add(replacement)
		break
	}
	newTx.Sent = false
	newTx.Data.GasFeeCap = newFeeCap
	newTx.Data.GasTipCap = newTipCap
	newTx.FullTx, err = p.auth.Signer(p.auth.From, types.NewTx(&newTx.Data))
	if err != nil {
		return err
	}

	return p.sendTx(ctx, prevTx, &newTx)
}

// the mutex must be held by the caller
func (p *DataPoster[Meta]) updateNonce(ctx context.Context) error {
	var blockNumQuery *big.Int
	if p.config().WaitForL1Finality {
		blockNumQuery = big.NewInt(int64(rpc.FinalizedBlockNumber))
	}
	header, err := p.client.HeaderByNumber(ctx, blockNumQuery)
	if err != nil {
		return fmt.Errorf("failed to get the latest or finalized L1 header: %w", err)
	}
	if p.lastBlock != nil && arbmath.BigEquals(p.lastBlock, header.Number) {
		return nil
	}
	nonce, err := p.client.NonceAt(ctx, p.auth.From, header.Number)
	if err != nil {
		if p.lastBlock != nil {
			log.Warn("failed to get current nonce", "lastBlock", p.lastBlock, "newBlock", header.Number, "err", err)
			return nil
		}
		return err
	}
	if nonce > p.nonce {
		log.Info("data poster transactions confirmed", "previousNonce", p.nonce, "newNonce", nonce, "previousL1Block", p.lastBlock, "newL1Block", header.Number)
		if len(p.errorCount) > 0 {
			for x := p.nonce; x < nonce; x++ {
				delete(p.errorCount, x)
			}
		}
		err := p.queue.Prune(ctx, nonce)
		if err != nil {
			return err
		}
	}
	// We update these two variables together because they should remain in sync even if there's an error.
	p.lastBlock = header.Number
	p.nonce = nonce
	return nil
}

func (p *DataPoster[Meta]) updateBalance(ctx context.Context) error {
	// Use the pending (representated as -1) balance because we're looking at batches we'd post,
	// so we want to see how much gas we could afford with our pending state.
	balance, err := p.client.BalanceAt(ctx, p.auth.From, big.NewInt(-1))
	if err != nil {
		return err
	}
	p.balance = balance
	return nil
}

const maxConsecutiveIntermittentErrors = 10

func (p *DataPoster[Meta]) maybeLogError(err error, tx *queuedTransaction[Meta], msg string) {
	nonce := tx.Data.Nonce
	if err == nil {
		delete(p.errorCount, nonce)
		return
	}
	logLevel := log.Error
	if errors.Is(err, ErrStorageRace) {
		p.errorCount[nonce]++
		if p.errorCount[nonce] <= maxConsecutiveIntermittentErrors {
			logLevel = log.Debug
		}
	} else {
		delete(p.errorCount, nonce)
	}
	logLevel(msg, "err", err, "nonce", nonce, "feeCap", tx.Data.GasFeeCap, "tipCap", tx.Data.GasTipCap)
}

const minWait = time.Second * 10

func (p *DataPoster[Meta]) Start(ctxIn context.Context) {
	p.StopWaiter.Start(ctxIn, p)
	p.CallIteratively(func(ctx context.Context) time.Duration {
		p.mutex.Lock()
		defer p.mutex.Unlock()
		if !p.redisLock.AttemptLock(ctx) {
			return p.replacementTimes[0]
		}
		err := p.updateBalance(ctx)
		if err != nil {
			log.Warn("failed to update tx poster balance", "err", err)
			return minWait
		}
		err = p.updateNonce(ctx)
		if err != nil {
			// This is non-fatal because it's only needed for clearing out old queue items.
			log.Warn("failed to update tx poster nonce", "err", err)
		}
		now := time.Now()
		nextCheck := now.Add(p.replacementTimes[0])
		maxTxsToRbf := p.config().MaxMempoolTransactions
		if maxTxsToRbf == 0 {
			maxTxsToRbf = 512
		}
		unconfirmedNonce, err := p.client.NonceAt(ctx, p.auth.From, nil)
		if err != nil {
			log.Warn("failed to get latest nonce", "err", err)
			return minWait
		}
		// We use unconfirmedNonce here to replace-by-fee transactions that aren't in a block,
		// excluding those that are in an unconfirmed block. If a reorg occurs, we'll continue
		// replacing them by fee.
		queueContents, err := p.queue.GetContents(ctx, unconfirmedNonce, maxTxsToRbf)
		if err != nil {
			log.Warn("failed to get tx queue contents", "err", err)
			return minWait
		}
		for index, tx := range queueContents {
			backlogOfBatches := len(queueContents) - index - 1
			replacing := false
			if now.After(tx.NextReplacement) {
				replacing = true
				err := p.replaceTx(ctx, tx, uint64(backlogOfBatches))
				p.maybeLogError(err, tx, "failed to replace-by-fee transaction")
			}
			if nextCheck.After(tx.NextReplacement) {
				nextCheck = tx.NextReplacement
			}
			if !replacing && !tx.Sent {
				err := p.sendTx(ctx, tx, tx)
				p.maybeLogError(err, tx, "failed to re-send transaction")
				if err != nil {
					nextSend := time.Now().Add(time.Minute)
					if nextCheck.After(nextSend) {
						nextCheck = nextSend
					}
				}
			}
		}
		wait := time.Until(nextCheck)
		if wait < minWait {
			wait = minWait
		}
		return wait
	})
}
