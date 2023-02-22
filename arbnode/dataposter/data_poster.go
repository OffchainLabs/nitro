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
	"github.com/holiman/uint256"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/blobs"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/signature"
	"github.com/offchainlabs/nitro/util/stopwaiter"
	"github.com/protolambda/ztyp/view"
	flag "github.com/spf13/pflag"
)

type queuedTransaction[Meta any] struct {
	FullTx          *types.Transaction
	Data            types.TxData
	BlobData        types.TxWrapData
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
	RedisSigner           signature.SimpleHmacConfig `koanf:"redis-signer"`
	ReplacementTimes      string                     `koanf:"replacement-times"`
	WaitForL1Finality     bool                       `koanf:"wait-for-l1-finality" reload:"hot"`
	MaxQueuedTransactions uint64                     `koanf:"max-queued-transactions" reload:"hot"`
	TargetPriceGwei       float64                    `koanf:"target-price-gwei" reload:"hot"`
	UrgencyGwei           float64                    `koanf:"urgency-gwei" reload:"hot"`
}

type DataPosterConfigFetcher func() *DataPosterConfig

func DataPosterConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.String(prefix+".replacement-times", DefaultDataPosterConfig.ReplacementTimes, "comma-separated list of durations since first posting to attempt a replace-by-fee")
	f.Bool(prefix+".wait-for-l1-finality", DefaultDataPosterConfig.WaitForL1Finality, "only treat a transaction as confirmed after L1 finality has been achieved (recommended)")
	f.Uint64(prefix+".max-queued-transactions", DefaultDataPosterConfig.MaxQueuedTransactions, "the maximum number of transactions to have queued in the mempool at once (0 = unlimited)")
	f.Float64(prefix+".target-price-gwei", DefaultDataPosterConfig.TargetPriceGwei, "the target price to use for maximum fee cap calculation")
	f.Float64(prefix+".urgency-gwei", DefaultDataPosterConfig.UrgencyGwei, "the urgency to use for maximum fee cap calculation")
	signature.SimpleHmacConfigAddOptions(prefix+".redis-signer", f)
}

var DefaultDataPosterConfig = DataPosterConfig{
	ReplacementTimes:      "5m,10m,20m,30m,1h,2h,4h,6h,8h,12h,16h,18h,20h,22h",
	WaitForL1Finality:     true,
	TargetPriceGwei:       60.,
	UrgencyGwei:           2.,
	MaxQueuedTransactions: 64,
}

var TestDataPosterConfig = DataPosterConfig{
	ReplacementTimes:      "1s,2s,5s,10s,20s,30s,1m,5m",
	RedisSigner:           signature.TestSimpleHmacConfig,
	WaitForL1Finality:     false,
	TargetPriceGwei:       60.,
	UrgencyGwei:           2.,
	MaxQueuedTransactions: 64,
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
	isEip4844         bool

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

func NewDataPoster[Meta any](headerReader *headerreader.HeaderReader, auth *bind.TransactOpts, redisClient redis.UniversalClient, redisLock AttemptLocker, config DataPosterConfigFetcher, isEip4844 bool, metadataRetriever func(ctx context.Context, blockNum *big.Int) (Meta, error)) (*DataPoster[Meta], error) {
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
		isEip4844:         isEip4844,
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
		maxQueueItems := p.config().MaxQueuedTransactions
		nextNonce := lastQueueItem.FullTx.Nonce() + 1
		if maxQueueItems > 0 && nextNonce >= p.nonce+maxQueueItems {
			return 0, emptyMeta, fmt.Errorf("attempting to post a transaction with nonce %v while current nonce is %v would exceed max data poster queue length of %v", nextNonce, p.nonce, maxQueueItems)
		}
		return nextNonce, lastQueueItem.Meta, nil
	}
	meta, err := p.metadataRetriever(ctx, p.lastBlock)
	return p.nonce, meta, err
}

const minRbfIncrease = arbmath.OneInBips * 11 / 10

func (p *DataPoster[Meta]) getFeeAndTipCaps(ctx context.Context, gasLimit uint64, lastTipCap *big.Int, dataCreatedAt time.Time, backlogOfBatches uint64) (*big.Int, *big.Int, error) {
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

// DataToPost defines a struct containing sequencer inbox calldata, which will be sent in a transaction
// to the sequencer inbox and an optional set of L2-specific message data which can be sent via an
// EIP-4844 style, shard blob transaction instead of calldata to L1 to save on costs.
type DataToPost struct {
	SequencerInboxCalldata []byte
	L2MessageData          []byte
}

func (p *DataPoster[Meta]) PostTransaction(ctx context.Context, dataCreatedAt time.Time, nonce uint64, meta Meta, to common.Address, data *DataToPost, gasLimit uint64) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	feeCap, tipCap, err := p.getFeeAndTipCaps(ctx, gasLimit, nil, dataCreatedAt, 0)
	if err != nil {
		return err
	}
	tx, txData, txWrapData, err := p.prepareTxTypeToPost(feeCap, tipCap, data, nonce, to, gasLimit)
	if err != nil {
		return err
	}
	fullTx, err := p.auth.Signer(p.auth.From, tx)
	if err != nil {
		return err
	}
	queuedTx := queuedTransaction[Meta]{
		Data:            txData,
		BlobData:        txWrapData,
		FullTx:          fullTx,
		Meta:            meta,
		Sent:            false,
		Created:         dataCreatedAt,
		NextReplacement: time.Now().Add(p.replacementTimes[0]),
	}
	return p.sendTx(ctx, nil, &queuedTx)
}

// Prepares a transaction kind to post depending on configuration values. This allows for
// posting EIP-4844 style blob transactions to reduce costs on L1.
func (p *DataPoster[Meta]) prepareTxTypeToPost(
	feeCap, tipCap *big.Int, data *DataToPost, nonce uint64, to common.Address, gasLimit uint64,
) (*types.Transaction, types.TxData, types.TxWrapData, error) {
	if p.isEip4844 {
		dataBlobs := blobs.EncodeBlobs(data.L2MessageData)
		commitments, versionedHashes, aggregatedProof, err := dataBlobs.ComputeCommitmentsAndAggregatedProof()
		if err != nil {
			return nil, nil, nil, err
		}
		tCap, overflows := uint256.FromBig(tipCap)
		if overflows {
			return nil, nil, nil, fmt.Errorf("tip cap overflows: %s", tipCap.String())
		}
		fCap, overflows := uint256.FromBig(feeCap)
		if overflows {
			return nil, nil, nil, fmt.Errorf("fee cap overflows: %s", feeCap.String())
		}
		txData := &types.SignedBlobTx{
			Message: types.BlobTxMessage{
				Nonce:               view.Uint64View(nonce),
				GasTipCap:           view.Uint256View(*tCap),
				GasFeeCap:           view.Uint256View(*fCap),
				Gas:                 view.Uint64View(gasLimit),
				Data:                data.SequencerInboxCalldata,
				To:                  types.AddressOptionalSSZ{Address: (*types.AddressSSZ)(&to)},
				Value:               view.Uint256View(*uint256.NewInt(0)),
				BlobVersionedHashes: versionedHashes,
				MaxFeePerDataGas:    view.Uint256View(*fCap), // Use the same fee cap as gas for now.
			},
		}
		blobWrap := &blobs.BlobTxWrapData{
			BlobKzgs:           commitments,
			Blobs:              dataBlobs,
			KzgAggregatedProof: aggregatedProof,
		}
		txWrapData := blobs.ToGethWrapData(blobWrap)
		return types.NewTx(txData, types.WithTxWrapData(txWrapData)), txData, txWrapData, nil
	}
	txData := &types.DynamicFeeTx{
		Nonce:     nonce,
		GasTipCap: tipCap,
		GasFeeCap: feeCap,
		Gas:       gasLimit,
		To:        &to,
		Value:     new(big.Int),
		Data:      data.SequencerInboxCalldata,
	}
	return types.NewTx(txData), txData, nil, nil
}

// the mutex must be held by the caller
func (p *DataPoster[Meta]) saveTx(ctx context.Context, prevTx *queuedTransaction[Meta], newTx *queuedTransaction[Meta]) error {
	if prevTx != nil && prevTx.FullTx.Nonce() != newTx.FullTx.Nonce() {
		return fmt.Errorf("prevTx nonce %v doesn't match newTx nonce %v", prevTx.FullTx.Nonce(), newTx.FullTx.Nonce())
	}
	return p.queue.Put(ctx, newTx.FullTx.Nonce(), prevTx, newTx)
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
func (p *DataPoster[Meta]) replaceTx(ctx context.Context, prevTx *queuedTransaction[Meta], backlogOfBatches uint64) error {
	newFeeCap, newTipCap, err := p.getFeeAndTipCaps(ctx, prevTx.FullTx.Gas(), prevTx.FullTx.GasTipCap(), prevTx.Created, backlogOfBatches)
	if err != nil {
		return err
	}

	desiredFeeCap := newFeeCap
	maxFeeCap := new(big.Int).Div(p.balance, new(big.Int).SetUint64(prevTx.FullTx.Gas()))
	newFeeCap = arbmath.BigMin(newFeeCap, maxFeeCap)
	minNewFeeCap := arbmath.BigMulByBips(prevTx.FullTx.GasFeeCap(), minRbfIncrease)
	newTx := *prevTx
	if newFeeCap.Cmp(minNewFeeCap) < 0 {
		if desiredFeeCap.Cmp(minNewFeeCap) >= 0 {
			log.Error(
				"lack of L1 balance prevents posting transaction with a higher fee cap",
				"balance", p.balance,
				"gasLimit", prevTx.FullTx.Gas(),
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
	var txData types.TxData
	var tx *types.Transaction
	if p.isEip4844 {
		tCap, ok := uint256.FromBig(newTipCap)
		if !ok {
			return errors.New("tip cap is not a big int")
		}
		fCap, ok := uint256.FromBig(newFeeCap)
		if !ok {
			return errors.New("fee cap is not a big int")
		}
		txData = &types.SignedBlobTx{
			Message: types.BlobTxMessage{
				Nonce:               view.Uint64View(newTx.FullTx.Nonce()),
				GasTipCap:           view.Uint256View(*tCap),
				GasFeeCap:           view.Uint256View(*fCap),
				Gas:                 view.Uint64View(newTx.FullTx.Gas()),
				To:                  types.AddressOptionalSSZ{Address: (*types.AddressSSZ)(newTx.FullTx.To())},
				Value:               view.Uint256View(*uint256.NewInt(0)),
				BlobVersionedHashes: newTx.FullTx.DataHashes(),
				MaxFeePerDataGas:    view.Uint256View(*fCap), // Use the same fee cap as gas for now.
			},
		}
		tx = types.NewTx(newTx.Data, types.WithTxWrapData(newTx.BlobData))
	} else {
		txData = &types.DynamicFeeTx{
			Nonce:     newTx.FullTx.Nonce(),
			GasTipCap: newTipCap,
			GasFeeCap: newFeeCap,
			Gas:       newTx.FullTx.Gas(),
			To:        newTx.FullTx.To(),
			Value:     newTx.FullTx.Value(),
			Data:      newTx.FullTx.Data(),
		}
		tx = types.NewTx(txData)
	}
	newTx.Sent = false
	newTx.Data = txData
	newTx.FullTx, err = p.auth.Signer(p.auth.From, tx)
	if err != nil {
		return err
	}

	return p.sendTx(ctx, prevTx, &newTx)
}

// the mutex must be held by the caller
func (p *DataPoster[Meta]) updateState(ctx context.Context) error {
	var blockNumQuery *big.Int
	if p.config().WaitForL1Finality {
		blockNumQuery = big.NewInt(int64(rpc.FinalizedBlockNumber))
	}
	header, err := p.client.HeaderByNumber(ctx, blockNumQuery)
	if err != nil {
		return err
	}
	p.lastBlock = header.Number
	nonce, err := p.client.NonceAt(ctx, p.auth.From, p.lastBlock)
	if err != nil {
		return err
	}
	if nonce > p.nonce {
		log.Info("data poster transactions confirmed", "previousNonce", p.nonce, "newNonce", nonce, "l1Block", p.lastBlock)
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
	nonce := tx.FullTx.Nonce()
	if err == nil {
		delete(p.errorCount, nonce)
		return
	}
	if errors.Is(err, ErrStorageRace) {
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
		maxTxsToRbf := p.config().MaxQueuedTransactions
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
