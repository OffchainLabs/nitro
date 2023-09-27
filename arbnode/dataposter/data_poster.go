// Copyright 2021-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

// Package dataposter implements generic functionality to post transactions.
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
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/go-redis/redis/v8"
	"github.com/offchainlabs/nitro/arbnode/dataposter/dbstorage"
	"github.com/offchainlabs/nitro/arbnode/dataposter/noop"
	"github.com/offchainlabs/nitro/arbnode/dataposter/slice"
	"github.com/offchainlabs/nitro/arbnode/dataposter/storage"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/signature"
	"github.com/offchainlabs/nitro/util/stopwaiter"
	"github.com/spf13/pflag"

	redisstorage "github.com/offchainlabs/nitro/arbnode/dataposter/redis"
)

// Dataposter implements functionality to post transactions on the chain. It
// is initialized with specified sender/signer and keeps nonce of that address
// as it posts transactions.
// Transactions are also saved in the queue when it's being sent, and when
// persistant storage is used for the queue, after restarting the node
// dataposter will pick up where it left.
// DataPoster must be RLP serializable and deserializable
type DataPoster struct {
	stopwaiter.StopWaiter
	headerReader      *headerreader.HeaderReader
	client            arbutil.L1Interface
	sender            common.Address
	signer            bind.SignerFn
	redisLock         AttemptLocker
	config            ConfigFetcher
	replacementTimes  []time.Duration
	metadataRetriever func(ctx context.Context, blockNum *big.Int) ([]byte, error)

	// These fields are protected by the mutex.
	// TODO: factor out these fields into separate structure, since now one
	// needs to make sure call sites of methods that change these values hold
	// the lock (currently ensured by having comments like:
	// "the mutex must be held by the caller" above the function).
	mutex      sync.Mutex
	lastBlock  *big.Int
	balance    *big.Int
	nonce      uint64
	queue      QueueStorage
	errorCount map[uint64]int // number of consecutive intermittent errors rbf-ing or sending, per nonce
}

type AttemptLocker interface {
	AttemptLock(context.Context) bool
}

func parseReplacementTimes(val string) ([]time.Duration, error) {
	var res []time.Duration
	var lastReplacementTime time.Duration
	for _, s := range strings.Split(val, ",") {
		t, err := time.ParseDuration(s)
		if err != nil {
			return nil, fmt.Errorf("parsing durations: %w", err)
		}
		if t <= lastReplacementTime {
			return nil, errors.New("replacement times must be increasing")
		}
		res = append(res, t)
		lastReplacementTime = t
	}
	if len(res) == 0 {
		log.Warn("disabling replace-by-fee for data poster")
	}
	// To avoid special casing "don't replace again", replace in 10 years.
	return append(res, time.Hour*24*365*10), nil
}

func NewDataPoster(ctx context.Context, db ethdb.Database, headerReader *headerreader.HeaderReader, auth *bind.TransactOpts, redisClient redis.UniversalClient, redisLock AttemptLocker, config ConfigFetcher, metadataRetriever func(ctx context.Context, blockNum *big.Int) ([]byte, error)) (*DataPoster, error) {
	initConfig := config()
	replacementTimes, err := parseReplacementTimes(initConfig.ReplacementTimes)
	if err != nil {
		return nil, err
	}
	if headerReader.IsParentChainArbitrum() && !initConfig.UseNoOpStorage {
		initConfig.UseNoOpStorage = true
		log.Info("Disabling data poster storage, as parent chain appears to be an Arbitrum chain without a mempool")
	}
	encF := func() storage.EncoderDecoderInterface {
		if config().LegacyStorageEncoding {
			return &storage.LegacyEncoderDecoder{}
		}
		return &storage.EncoderDecoder{}
	}
	var queue QueueStorage
	switch {
	case initConfig.UseNoOpStorage:
		queue = &noop.Storage{}
	case redisClient != nil:
		var err error
		queue, err = redisstorage.NewStorage(redisClient, "data-poster.queue", &initConfig.RedisSigner, encF)
		if err != nil {
			return nil, err
		}
	case initConfig.UseDBStorage:
		storage := dbstorage.New(db, func() storage.EncoderDecoderInterface { return &storage.EncoderDecoder{} })
		if config().Dangerous.ClearDBStorage {
			if err := storage.PruneAll(ctx); err != nil {
				return nil, err
			}
		}
		queue = storage
	default:
		queue = slice.NewStorage(func() storage.EncoderDecoderInterface { return &storage.EncoderDecoder{} })
	}
	return &DataPoster{
		headerReader:      headerReader,
		client:            headerReader.Client(),
		sender:            auth.From,
		signer:            auth.Signer,
		config:            config,
		replacementTimes:  replacementTimes,
		metadataRetriever: metadataRetriever,
		queue:             queue,
		redisLock:         redisLock,
		errorCount:        make(map[uint64]int),
	}, nil
}

func (p *DataPoster) Sender() common.Address {
	return p.sender
}

// Does basic check whether posting transaction with specified nonce would
// result in exceeding maximum queue length or maximum transactions in mempool.
func (p *DataPoster) canPostWithNonce(ctx context.Context, nextNonce uint64) error {
	cfg := p.config()
	// If the queue has reached configured max size, don't post a transaction.
	if cfg.MaxQueuedTransactions > 0 {
		queueLen, err := p.queue.Length(ctx)
		if err != nil {
			return fmt.Errorf("getting queue length: %w", err)
		}
		if queueLen >= cfg.MaxQueuedTransactions {
			return fmt.Errorf("posting a transaction with nonce: %d will exceed max allowed dataposter queued transactions: %d, current nonce: %d", nextNonce, cfg.MaxQueuedTransactions, p.nonce)
		}
	}
	// Check that posting a new transaction won't exceed maximum pending
	// transactions in mempool.
	if cfg.MaxMempoolTransactions > 0 {
		unconfirmedNonce, err := p.client.NonceAt(ctx, p.sender, nil)
		if err != nil {
			return fmt.Errorf("getting nonce of a dataposter sender: %w", err)
		}
		if nextNonce >= cfg.MaxMempoolTransactions+unconfirmedNonce {
			return fmt.Errorf("posting a transaction with nonce: %d will exceed max mempool size: %d, unconfirmed nonce: %d", nextNonce, cfg.MaxMempoolTransactions, unconfirmedNonce)
		}
	}
	return nil
}

func (p *DataPoster) waitForL1Finality() bool {
	return p.config().WaitForL1Finality && !p.headerReader.IsParentChainArbitrum()
}

// Requires the caller hold the mutex.
// Returns the next nonce, its metadata if stored, a bool indicating if the metadata is present, and an error.
// Unlike GetNextNonceAndMeta, this does not call the metadataRetriever if the metadata is not stored in the queue.
func (p *DataPoster) getNextNonceAndMaybeMeta(ctx context.Context) (uint64, []byte, bool, error) {
	// Ensure latest finalized block state is available.
	blockNum, err := p.client.BlockNumber(ctx)
	if err != nil {
		return 0, nil, false, err
	}
	lastQueueItem, err := p.queue.FetchLast(ctx)
	if err != nil {
		return 0, nil, false, fmt.Errorf("fetching last element from queue: %w", err)
	}
	if lastQueueItem != nil {
		nextNonce := lastQueueItem.Data.Nonce + 1
		if err := p.canPostWithNonce(ctx, nextNonce); err != nil {
			return 0, nil, false, err
		}
		return nextNonce, lastQueueItem.Meta, true, nil
	}

	if err := p.updateNonce(ctx); err != nil {
		if !p.queue.IsPersistent() && p.waitForL1Finality() {
			return 0, nil, false, fmt.Errorf("error getting latest finalized nonce (and queue is not persistent): %w", err)
		}
		// Fall back to using a recent block to get the nonce. This is safe because there's nothing in the queue.
		nonceQueryBlock := arbmath.UintToBig(arbmath.SaturatingUSub(blockNum, 1))
		log.Warn("failed to update nonce with queue empty; falling back to using a recent block", "recentBlock", nonceQueryBlock, "err", err)
		nonce, err := p.client.NonceAt(ctx, p.sender, nonceQueryBlock)
		if err != nil {
			return 0, nil, false, fmt.Errorf("failed to get nonce at block %v: %w", nonceQueryBlock, err)
		}
		p.lastBlock = nonceQueryBlock
		p.nonce = nonce
	}
	return p.nonce, nil, false, nil
}

// GetNextNonceAndMeta retrieves generates next nonce, validates that a
// transaction can be posted with that nonce, and fetches "Meta" either last
// queued iterm (if queue isn't empty) or retrieves with last block.
func (p *DataPoster) GetNextNonceAndMeta(ctx context.Context) (uint64, []byte, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	nonce, meta, hasMeta, err := p.getNextNonceAndMaybeMeta(ctx)
	if err != nil {
		return 0, nil, err
	}
	if !hasMeta {
		meta, err = p.metadataRetriever(ctx, p.lastBlock)
	}
	return nonce, meta, err
}

const minRbfIncrease = arbmath.OneInBips * 11 / 10

func (p *DataPoster) feeAndTipCaps(ctx context.Context, nonce uint64, gasLimit uint64, lastFeeCap *big.Int, lastTipCap *big.Int, dataCreatedAt time.Time, backlogOfBatches uint64) (*big.Int, *big.Int, error) {
	config := p.config()
	latestHeader, err := p.headerReader.LastHeader(ctx)
	if err != nil {
		return nil, nil, err
	}
	if latestHeader.BaseFee == nil {
		return nil, nil, fmt.Errorf("latest parent chain block %v missing BaseFee (either the parent chain does not have EIP-1559 or the parent chain node is not synced)", latestHeader.Number)
	}
	softConfBlock := arbmath.BigSubByUint(latestHeader.Number, config.NonceRbfSoftConfs)
	softConfNonce, err := p.client.NonceAt(ctx, p.sender, softConfBlock)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get latest nonce %v blocks ago (block %v): %w", config.NonceRbfSoftConfs, softConfBlock, err)
	}
	newFeeCap := new(big.Int).Mul(latestHeader.BaseFee, big.NewInt(2))
	newFeeCap = arbmath.BigMax(newFeeCap, arbmath.FloatToBig(config.MinFeeCapGwei*params.GWei))

	newTipCap, err := p.client.SuggestGasTipCap(ctx)
	if err != nil {
		return nil, nil, err
	}
	newTipCap = arbmath.BigMax(newTipCap, arbmath.FloatToBig(config.MinTipCapGwei*params.GWei))
	newTipCap = arbmath.BigMin(newTipCap, arbmath.FloatToBig(config.MaxTipCapGwei*params.GWei))

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

	latestBalance := p.balance
	balanceForTx := new(big.Int).Set(latestBalance)
	if config.AllocateMempoolBalance && !config.UseNoOpStorage {
		// We reserve half the balance for the first transaction, and then split the remaining balance for all after that.
		// With noop storage, we don't try to replace-by-fee, so we don't need to worry about this.
		balanceForTx.Div(balanceForTx, common.Big2)
		if nonce != softConfNonce && config.MaxMempoolTransactions > 1 {
			// balanceForTx /= config.MaxMempoolTransactions-1
			balanceForTx.Div(balanceForTx, arbmath.UintToBig(config.MaxMempoolTransactions-1))
		}
	}
	balanceFeeCap := arbmath.BigDivByUint(balanceForTx, gasLimit)
	if arbmath.BigGreaterThan(newFeeCap, balanceFeeCap) {
		log.Error(
			"lack of L1 balance prevents posting transaction with desired fee cap",
			"balance", latestBalance,
			"maxTransactions", config.MaxMempoolTransactions,
			"balanceForTransaction", balanceForTx,
			"gasLimit", gasLimit,
			"desiredFeeCap", newFeeCap,
			"balanceFeeCap", balanceFeeCap,
			"nonce", nonce,
			"softConfNonce", softConfNonce,
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

func (p *DataPoster) PostTransaction(ctx context.Context, dataCreatedAt time.Time, nonce uint64, meta []byte, to common.Address, calldata []byte, gasLimit uint64, value *big.Int) (*types.Transaction, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	expectedNonce, _, _, err := p.getNextNonceAndMaybeMeta(ctx)
	if err != nil {
		return nil, err
	}
	if nonce != expectedNonce {
		return nil, fmt.Errorf("data poster expected next transaction to have nonce %v but was requested to post transaction with nonce %v", expectedNonce, nonce)
	}

	err = p.updateBalance(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to update data poster balance: %w", err)
	}

	feeCap, tipCap, err := p.feeAndTipCaps(ctx, nonce, gasLimit, nil, nil, dataCreatedAt, 0)
	if err != nil {
		return nil, err
	}
	inner := types.DynamicFeeTx{
		Nonce:     nonce,
		GasTipCap: tipCap,
		GasFeeCap: feeCap,
		Gas:       gasLimit,
		To:        &to,
		Value:     value,
		Data:      calldata,
	}
	fullTx, err := p.signer(p.sender, types.NewTx(&inner))
	if err != nil {
		return nil, fmt.Errorf("signing transaction: %w", err)
	}
	queuedTx := storage.QueuedTransaction{
		Data:            inner,
		FullTx:          fullTx,
		Meta:            meta,
		Sent:            false,
		Created:         dataCreatedAt,
		NextReplacement: time.Now().Add(p.replacementTimes[0]),
	}
	return fullTx, p.sendTx(ctx, nil, &queuedTx)
}

// the mutex must be held by the caller
func (p *DataPoster) saveTx(ctx context.Context, prevTx, newTx *storage.QueuedTransaction) error {
	if prevTx != nil && prevTx.Data.Nonce != newTx.Data.Nonce {
		return fmt.Errorf("prevTx nonce %v doesn't match newTx nonce %v", prevTx.Data.Nonce, newTx.Data.Nonce)
	}
	if err := p.queue.Put(ctx, newTx.Data.Nonce, prevTx, newTx); err != nil {
		return fmt.Errorf("putting new tx in the queue: %w", err)
	}
	return nil
}

func (p *DataPoster) sendTx(ctx context.Context, prevTx *storage.QueuedTransaction, newTx *storage.QueuedTransaction) error {
	if prevTx == nil || (newTx.FullTx.Hash() != prevTx.FullTx.Hash()) {
		if err := p.saveTx(ctx, prevTx, newTx); err != nil {
			return err
		}
	}
	if err := p.client.SendTransaction(ctx, newTx.FullTx); err != nil {
		if !strings.Contains(err.Error(), "already known") && !strings.Contains(err.Error(), "nonce too low") {
			log.Warn("DataPoster failed to send transaction", "err", err, "nonce", newTx.FullTx.Nonce(), "feeCap", newTx.FullTx.GasFeeCap(), "tipCap", newTx.FullTx.GasTipCap())
			return err
		}
		log.Info("DataPoster transaction already known", "err", err, "nonce", newTx.FullTx.Nonce(), "hash", newTx.FullTx.Hash())
	} else {
		log.Info("DataPoster sent transaction", "nonce", newTx.FullTx.Nonce(), "hash", newTx.FullTx.Hash(), "feeCap", newTx.FullTx.GasFeeCap())
	}
	newerTx := *newTx
	newerTx.Sent = true
	return p.saveTx(ctx, newTx, &newerTx)
}

// The mutex must be held by the caller.
func (p *DataPoster) replaceTx(ctx context.Context, prevTx *storage.QueuedTransaction, backlogOfBatches uint64) error {
	newFeeCap, newTipCap, err := p.feeAndTipCaps(ctx, prevTx.Data.Nonce, prevTx.Data.Gas, prevTx.Data.GasFeeCap, prevTx.Data.GasTipCap, prevTx.Created, backlogOfBatches)
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
	newTx.FullTx, err = p.signer(p.sender, types.NewTx(&newTx.Data))
	if err != nil {
		return err
	}

	return p.sendTx(ctx, prevTx, &newTx)
}

// Gets latest known or finalized block header (depending on config flag),
// gets the nonce of the dataposter sender and stores it if it has increased.
// The mutex must be held by the caller.
func (p *DataPoster) updateNonce(ctx context.Context) error {
	var blockNumQuery *big.Int
	if p.waitForL1Finality() {
		blockNumQuery = big.NewInt(int64(rpc.FinalizedBlockNumber))
	}
	header, err := p.client.HeaderByNumber(ctx, blockNumQuery)
	if err != nil {
		return fmt.Errorf("failed to get the latest or finalized L1 header: %w", err)
	}
	if p.lastBlock != nil && arbmath.BigEquals(p.lastBlock, header.Number) {
		return nil
	}
	nonce, err := p.client.NonceAt(ctx, p.sender, header.Number)
	if err != nil {
		if p.lastBlock != nil {
			log.Warn("Failed to get current nonce", "lastBlock", p.lastBlock, "newBlock", header.Number, "err", err)
			return nil
		}
		return err
	}
	// Ignore if nonce hasn't increased.
	if nonce <= p.nonce {
		// Still update last block number.
		if nonce == p.nonce {
			p.lastBlock = header.Number
		}
		return nil
	}
	log.Info("Data poster transactions confirmed", "previousNonce", p.nonce, "newNonce", nonce, "previousL1Block", p.lastBlock, "newL1Block", header.Number)
	if len(p.errorCount) > 0 {
		for x := p.nonce; x < nonce; x++ {
			delete(p.errorCount, x)
		}
	}
	// We don't prune the most recent transaction in order to ensure that the data poster
	// always has a reference point in its queue of the latest transaction nonce and metadata.
	// nonce > 0 is implied by nonce > p.nonce, so this won't underflow.
	if err := p.queue.Prune(ctx, nonce-1); err != nil {
		return err
	}
	// We update these two variables together because they should remain in sync even if there's an error.
	p.lastBlock = header.Number
	p.nonce = nonce
	return nil
}

// Updates dataposter balance to balance at pending block.
func (p *DataPoster) updateBalance(ctx context.Context) error {
	// Use the pending (representated as -1) balance because we're looking at batches we'd post,
	// so we want to see how much gas we could afford with our pending state.
	balance, err := p.client.BalanceAt(ctx, p.sender, big.NewInt(-1))
	if err != nil {
		return err
	}
	p.balance = balance
	return nil
}

const maxConsecutiveIntermittentErrors = 10

func (p *DataPoster) maybeLogError(err error, tx *storage.QueuedTransaction, msg string) {
	nonce := tx.Data.Nonce
	if err == nil {
		delete(p.errorCount, nonce)
		return
	}
	logLevel := log.Error
	if errors.Is(err, storage.ErrStorageRace) {
		p.errorCount[nonce]++
		if p.errorCount[nonce] <= maxConsecutiveIntermittentErrors {
			logLevel = log.Debug
		}
	} else {
		delete(p.errorCount, nonce)
	}
	logLevel(msg, "err", err, "nonce", nonce, "feeCap", tx.Data.GasFeeCap, "tipCap", tx.Data.GasTipCap, "gas", tx.Data.Gas)
}

const minWait = time.Second * 10

// Tries to acquire redis lock, updates balance and nonce,
func (p *DataPoster) Start(ctxIn context.Context) {
	p.StopWaiter.Start(ctxIn, p)
	p.CallIteratively(func(ctx context.Context) time.Duration {
		p.mutex.Lock()
		defer p.mutex.Unlock()
		if !p.redisLock.AttemptLock(ctx) {
			return minWait
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
		unconfirmedNonce, err := p.client.NonceAt(ctx, p.sender, nil)
		if err != nil {
			log.Warn("Failed to get latest nonce", "err", err)
			return minWait
		}
		// We use unconfirmedNonce here to replace-by-fee transactions that aren't in a block,
		// excluding those that are in an unconfirmed block. If a reorg occurs, we'll continue
		// replacing them by fee.
		queueContents, err := p.queue.FetchContents(ctx, unconfirmedNonce, maxTxsToRbf)
		if err != nil {
			log.Error("Failed to fetch tx queue contents", "err", err)
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

// Implements queue-alike storage that can
// - Insert item at specified index
// - Update item with the condition that existing value equals assumed value
// - Delete all the items up to specified index (prune)
// - Calculate length
// Note: one of the implementation of this interface (Redis storage) does not
// support duplicate values.
type QueueStorage interface {
	// Returns at most maxResults items starting from specified index.
	FetchContents(ctx context.Context, startingIndex uint64, maxResults uint64) ([]*storage.QueuedTransaction, error)
	// Returns item with the biggest index.
	FetchLast(ctx context.Context) (*storage.QueuedTransaction, error)
	// Prunes items up to (excluding) specified index.
	Prune(ctx context.Context, until uint64) error
	// Inserts new item at specified index if previous value matches specified value.
	Put(ctx context.Context, index uint64, prevItem, newItem *storage.QueuedTransaction) error
	// Returns the size of a queue.
	Length(ctx context.Context) (int, error)
	// Indicates whether queue stored at disk.
	IsPersistent() bool
}

type DataPosterConfig struct {
	RedisSigner      signature.SimpleHmacConfig `koanf:"redis-signer"`
	ReplacementTimes string                     `koanf:"replacement-times"`
	// This is forcibly disabled if the parent chain is an Arbitrum chain,
	// so you should probably use DataPoster's waitForL1Finality method instead of reading this field directly.
	WaitForL1Finality      bool            `koanf:"wait-for-l1-finality" reload:"hot"`
	MaxMempoolTransactions uint64          `koanf:"max-mempool-transactions" reload:"hot"`
	MaxQueuedTransactions  int             `koanf:"max-queued-transactions" reload:"hot"`
	TargetPriceGwei        float64         `koanf:"target-price-gwei" reload:"hot"`
	UrgencyGwei            float64         `koanf:"urgency-gwei" reload:"hot"`
	MinFeeCapGwei          float64         `koanf:"min-fee-cap-gwei" reload:"hot"`
	MinTipCapGwei          float64         `koanf:"min-tip-cap-gwei" reload:"hot"`
	MaxTipCapGwei          float64         `koanf:"max-tip-cap-gwei" reload:"hot"`
	NonceRbfSoftConfs      uint64          `koanf:"nonce-rbf-soft-confs" reload:"hot"`
	AllocateMempoolBalance bool            `koanf:"allocate-mempool-balance" reload:"hot"`
	UseDBStorage           bool            `koanf:"use-db-storage"`
	UseNoOpStorage         bool            `koanf:"use-noop-storage"`
	LegacyStorageEncoding  bool            `koanf:"legacy-storage-encoding" reload:"hot"`
	Dangerous              DangerousConfig `koanf:"dangerous"`
}

type DangerousConfig struct {
	// This should be used with caution, only when dataposter somehow gets in a
	// bad state and we require clearing it.
	ClearDBStorage bool `koanf:"clear-dbstorage"`
}

// ConfigFetcher function type is used instead of directly passing config so
// that flags can be reloaded dynamically.
type ConfigFetcher func() *DataPosterConfig

func DataPosterConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.String(prefix+".replacement-times", DefaultDataPosterConfig.ReplacementTimes, "comma-separated list of durations since first posting to attempt a replace-by-fee")
	f.Bool(prefix+".wait-for-l1-finality", DefaultDataPosterConfig.WaitForL1Finality, "only treat a transaction as confirmed after L1 finality has been achieved (recommended)")
	f.Uint64(prefix+".max-mempool-transactions", DefaultDataPosterConfig.MaxMempoolTransactions, "the maximum number of transactions to have queued in the mempool at once (0 = unlimited)")
	f.Int(prefix+".max-queued-transactions", DefaultDataPosterConfig.MaxQueuedTransactions, "the maximum number of unconfirmed transactions to track at once (0 = unlimited)")
	f.Float64(prefix+".target-price-gwei", DefaultDataPosterConfig.TargetPriceGwei, "the target price to use for maximum fee cap calculation")
	f.Float64(prefix+".urgency-gwei", DefaultDataPosterConfig.UrgencyGwei, "the urgency to use for maximum fee cap calculation")
	f.Float64(prefix+".min-fee-cap-gwei", DefaultDataPosterConfig.MinFeeCapGwei, "the minimum fee cap to post transactions at")
	f.Float64(prefix+".min-tip-cap-gwei", DefaultDataPosterConfig.MinTipCapGwei, "the minimum tip cap to post transactions at")
	f.Float64(prefix+".max-tip-cap-gwei", DefaultDataPosterConfig.MaxTipCapGwei, "the maximum tip cap to post transactions at")
	f.Uint64(prefix+".nonce-rbf-soft-confs", DefaultDataPosterConfig.NonceRbfSoftConfs, "the maximum probable reorg depth, used to determine when a transaction will no longer likely need replaced-by-fee")
	f.Bool(prefix+".allocate-mempool-balance", DefaultDataPosterConfig.AllocateMempoolBalance, "if true, don't put transactions in the mempool that spend a total greater than the batch poster's balance")
	f.Bool(prefix+".use-db-storage", DefaultDataPosterConfig.UseDBStorage, "uses database storage when enabled")
	f.Bool(prefix+".use-noop-storage", DefaultDataPosterConfig.UseNoOpStorage, "uses noop storage, it doesn't store anything")
	f.Bool(prefix+".legacy-storage-encoding", DefaultDataPosterConfig.LegacyStorageEncoding, "encodes items in a legacy way (as it was before dropping generics)")

	signature.SimpleHmacConfigAddOptions(prefix+".redis-signer", f)
	addDangerousOptions(prefix+".dangerous", f)
}

func addDangerousOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".clear-dbstorage", DefaultDataPosterConfig.Dangerous.ClearDBStorage, "clear database storage")
}

var DefaultDataPosterConfig = DataPosterConfig{
	ReplacementTimes:       "5m,10m,20m,30m,1h,2h,4h,6h,8h,12h,16h,18h,20h,22h",
	WaitForL1Finality:      true,
	TargetPriceGwei:        60.,
	UrgencyGwei:            2.,
	MaxMempoolTransactions: 10,
	MinTipCapGwei:          0.05,
	MaxTipCapGwei:          5,
	NonceRbfSoftConfs:      1,
	AllocateMempoolBalance: true,
	UseDBStorage:           true,
	UseNoOpStorage:         false,
	LegacyStorageEncoding:  true,
	Dangerous:              DangerousConfig{ClearDBStorage: false},
}

var DefaultDataPosterConfigForValidator = func() DataPosterConfig {
	config := DefaultDataPosterConfig
	config.MaxMempoolTransactions = 1 // the validator cannot queue transactions
	return config
}()

var TestDataPosterConfig = DataPosterConfig{
	ReplacementTimes:       "1s,2s,5s,10s,20s,30s,1m,5m",
	RedisSigner:            signature.TestSimpleHmacConfig,
	WaitForL1Finality:      false,
	TargetPriceGwei:        60.,
	UrgencyGwei:            2.,
	MaxMempoolTransactions: 10,
	MinTipCapGwei:          0.05,
	MaxTipCapGwei:          5,
	NonceRbfSoftConfs:      1,
	AllocateMempoolBalance: true,
	UseDBStorage:           false,
	UseNoOpStorage:         false,
}

var TestDataPosterConfigForValidator = func() DataPosterConfig {
	config := TestDataPosterConfig
	config.MaxMempoolTransactions = 1 // the validator cannot queue transactions
	return config
}()
