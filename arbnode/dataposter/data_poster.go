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
}

type DataPosterConfig struct {
	RedisSigner       signature.SimpleHmacConfig `koanf:"redis-signer"`
	ReplacementTimes  string                     `koanf:"replacement-times"`
	L1LookBehind      uint64                     `koanf:"l1-look-behind" reload:"hot"`
	MaxFeeCapGwei     float64                    `koanf:"max-fee-cap-gwei" reload:"hot"`
	MaxFeeCapDoubling time.Duration              `koanf:"max-fee-cap-doubling" reload:"hot"`
}

type DataPosterConfigFetcher func() *DataPosterConfig

func DataPosterConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.String(prefix+".replacement-times", DefaultDataPosterConfig.ReplacementTimes, "comma-separated list of durations since first posting to attempt a replace-by-fee")
	f.Uint64(prefix+".l1-look-behind", DefaultDataPosterConfig.L1LookBehind, "look at state this many blocks behind the latest (fixes L1 node inconsistencies)")
	f.Float64(prefix+".max-fee-cap-gwei", DefaultDataPosterConfig.MaxFeeCapGwei, "the maximum fee cap to use, doubled every max-fee-cap-doubling")
	f.Duration(prefix+".max-fee-cap-doubling", DefaultDataPosterConfig.MaxFeeCapDoubling, "after this duration, double the fee cap (repeats)")
	signature.SimpleHmacConfigAddOptions(prefix+".redis-signer", f)
}

var DefaultDataPosterConfig = DataPosterConfig{
	ReplacementTimes:  "5m,10m,20m,30m,1h,2h,4h,6h,8h,12h,16h,18h,20h,22h",
	L1LookBehind:      2,
	MaxFeeCapGwei:     100.,
	MaxFeeCapDoubling: 2 * time.Hour,
}

var TestDataPosterConfig = DataPosterConfig{
	ReplacementTimes:  "1s,2s,5s,10s,20s,30s,1m,5m",
	RedisSigner:       signature.TestSimpleHmacConfig,
	L1LookBehind:      0,
	MaxFeeCapGwei:     100.,
	MaxFeeCapDoubling: 5 * time.Second,
}

// Meta must be RLP serializable and deserializable
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
	var emptyMeta Meta
	p.mutex.Lock()
	defer p.mutex.Unlock()
	err := p.updateState(ctx)
	if err != nil {
		return 0, emptyMeta, err
	}
	lastQueueItem, err := p.queue.GetLast(ctx)
	if err != nil {
		return 0, emptyMeta, err
	}
	if lastQueueItem != nil {
		return lastQueueItem.Data.Nonce + 1, lastQueueItem.Meta, nil
	}
	meta, err := p.metadataRetriever(ctx, p.lastBlock)
	return p.nonce, meta, err
}

const minRbfIncrease arbmath.Bips = arbmath.OneInBips * 11 / 10

func (p *DataPoster[Meta]) getFeeAndTipCaps(ctx context.Context, lastTipCap *big.Int, dataCreatedAt time.Time) (*big.Int, *big.Int, error) {
	latestHeader, err := p.headerReader.LastHeader(ctx)
	if err != nil {
		return nil, nil, err
	}
	newTipCap, err := p.client.SuggestGasTipCap(ctx)
	if err != nil {
		return nil, nil, err
	}
	if lastTipCap != nil {
		newTipCap = arbmath.BigMax(newTipCap, arbmath.BigMulByBips(lastTipCap, minRbfIncrease))
	}
	newFeeCap := new(big.Int).Mul(latestHeader.BaseFee, big.NewInt(2))
	newFeeCap.Add(newFeeCap, newTipCap)

	elapsed := time.Since(dataCreatedAt)
	config := p.config()
	maxFeeCap := new(big.Int).SetUint64(uint64(config.MaxFeeCapGwei * params.GWei))
	maxFeeCapDoublings := int64(elapsed / config.MaxFeeCapDoubling)
	// in tests, this could get way too big
	if maxFeeCapDoublings > 8 {
		maxFeeCapDoublings = 8
	}
	multiplier := new(big.Int).Exp(big.NewInt(2), big.NewInt(maxFeeCapDoublings), nil)
	maxFeeCap.Mul(maxFeeCap, multiplier)
	if arbmath.BigGreaterThan(newFeeCap, maxFeeCap) {
		logLevel := log.Info
		if maxFeeCapDoublings >= 3 {
			logLevel = log.Error
		} else if maxFeeCapDoublings >= 1 {
			logLevel = log.Warn
		}
		logLevel(
			"reducing proposed fee cap to current maximum",
			"proposedFeeCap", newFeeCap,
			"maxFeeCap", maxFeeCap,
			"elapsed", elapsed,
		)
		newFeeCap = maxFeeCap
	}

	return newFeeCap, newTipCap, nil
}

func (p *DataPoster[Meta]) PostTransaction(ctx context.Context, dataCreatedAt time.Time, nonce uint64, meta Meta, to common.Address, calldata []byte, gasLimit uint64) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	feeCap, tipCap, err := p.getFeeAndTipCaps(ctx, nil, dataCreatedAt)
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
			log.Warn("DataPoster failed to send transaction", "err", err, "nonce", newTx.FullTx.Nonce(), "feeCap", newTx.FullTx.GasFeeCap())
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
func (p *DataPoster[Meta]) replaceTx(ctx context.Context, prevTx *queuedTransaction[Meta]) error {
	newFeeCap, newTipCap, err := p.getFeeAndTipCaps(ctx, prevTx.Data.GasTipCap, prevTx.Created)
	if err != nil {
		return err
	}

	desiredFeeCap := newFeeCap
	maxFeeCap := new(big.Int).Div(p.balance, new(big.Int).SetUint64(prevTx.Data.Gas))
	newFeeCap = arbmath.BigMin(newFeeCap, maxFeeCap)
	minNewFeeCap := arbmath.BigMulByBips(prevTx.Data.GasFeeCap, minRbfIncrease)
	newTx := *prevTx
	if newFeeCap.Cmp(minNewFeeCap) < 0 {
		if desiredFeeCap.Cmp(minNewFeeCap) >= 0 {
			log.Error(
				"lack of L1 balance prevents posting transaction with a higher fee cap",
				"balance", p.balance,
				"gasLimit", prevTx.Data.Gas,
				"desiredFeeCap", desiredFeeCap,
				"maxFeeCap", maxFeeCap,
			)
		}
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
func (p *DataPoster[Meta]) updateState(ctx context.Context) error {
	header, err := p.client.HeaderByNumber(ctx, nil)
	if err != nil {
		return err
	}
	p.lastBlock = arbmath.BigSub(header.Number, new(big.Int).SetUint64(p.config().L1LookBehind))
	nonce, err := p.client.NonceAt(ctx, p.auth.From, p.lastBlock)
	if err != nil {
		return err
	}
	if nonce > p.nonce {
		if len(p.errorCount) > 0 {
			for x := p.nonce; x < nonce; x++ {
				delete(p.errorCount, x)
			}
		}
		err := p.queue.Prune(ctx, nonce)
		if err != nil {
			return err
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

const maxConsecutiveIntermittentErrors = 10

func (p *DataPoster[Meta]) maybeLogError(err error, tx *queuedTransaction[Meta], msg string) {
	nonce := tx.Data.Nonce
	if err == nil {
		delete(p.errorCount, nonce)
		return
	}
	if errors.Is(err, StorageRaceErr) {
		p.errorCount[nonce]++
		if p.errorCount[nonce] <= maxConsecutiveIntermittentErrors {
			log.Debug(msg, "err", err, "nonce", nonce)
			return
		}
	} else {
		delete(p.errorCount, nonce)
	}
	log.Error(msg, "err", err, "nonce", nonce)
}

const minWait = time.Second * 10
const maxTxsToRbf = 256

func (p *DataPoster[Meta]) Start(ctxIn context.Context) {
	p.StopWaiter.Start(ctxIn, p)
	p.CallIteratively(func(ctx context.Context) time.Duration {
		p.mutex.Lock()
		defer p.mutex.Unlock()
		if !p.redisLock.AttemptLock(ctx) {
			return p.replacementTimes[0]
		}
		err := p.updateState(ctx)
		if err != nil {
			log.Warn("failed to update tx poster internal state", "err", err)
			return minWait
		}
		now := time.Now()
		nextCheck := now.Add(p.replacementTimes[0])
		queueContents, err := p.queue.GetContents(ctx, p.nonce, maxTxsToRbf)
		if err != nil {
			log.Warn("failed to get tx queue contents", "err", err)
			return minWait
		}
		for _, tx := range queueContents {
			replacing := false
			if now.After(tx.NextReplacement) {
				replacing = true
				err := p.replaceTx(ctx, tx)
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
