// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

// Package dataposter implements generic functionality to post transactions.
package dataposter

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/Knetic/govaluate"
	"github.com/holiman/uint256"
	"github.com/redis/go-redis/v9"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbnode/dataposter/config"
	"github.com/offchainlabs/nitro/arbnode/dataposter/dbstorage"
	"github.com/offchainlabs/nitro/arbnode/dataposter/externalsigner"
	"github.com/offchainlabs/nitro/arbnode/dataposter/fees"
	"github.com/offchainlabs/nitro/arbnode/dataposter/lifecycle"
	datapostermetrics "github.com/offchainlabs/nitro/arbnode/dataposter/metrics"
	"github.com/offchainlabs/nitro/arbnode/dataposter/noop"
	redisstorage "github.com/offchainlabs/nitro/arbnode/dataposter/redis"
	"github.com/offchainlabs/nitro/arbnode/dataposter/slice"
	"github.com/offchainlabs/nitro/arbnode/dataposter/state"
	"github.com/offchainlabs/nitro/arbnode/dataposter/storage"
	"github.com/offchainlabs/nitro/arbnode/parent"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/blobs"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/rpcclient"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

// DataPoster implements functionality to post transactions on the chain. It
// is initialized with specified sender/signer and keeps nonce of that address
// as it posts transactions.
// Transactions are also saved in the queue when it's being sent, and when
// persistent storage is used for the queue, after restarting the node
// dataposter will pick up where it left.
// DataPoster must be RLP serializable and deserializable
type DataPoster struct {
	stopwaiter.StopWaiter
	headerReader      *headerreader.HeaderReader
	client            *ethclient.Client
	auth              *bind.TransactOpts
	signer            externalsigner.SignerFn
	config            config.ConfigFetcher
	usingNoOpStorage  bool
	metadataRetriever func(ctx context.Context, blockNum *big.Int) ([]byte, error)
	extraBacklog      func() uint64
	parentChainID256  *uint256.Int
	parentChain       *parent.ParentChain

	internalState *state.InternalState

	maxFeeCapExpression *govaluate.EvaluableExpression
}

type DataPosterOpts struct {
	Database          ethdb.Database
	HeaderReader      *headerreader.HeaderReader
	Auth              *bind.TransactOpts
	RedisClient       redis.UniversalClient
	Config            config.ConfigFetcher
	MetadataRetriever func(ctx context.Context, blockNum *big.Int) ([]byte, error)
	ExtraBacklog      func() uint64
	RedisKey          string // Redis storage key
	ParentChain       *parent.ParentChain
}

func NewDataPoster(ctx context.Context, opts *DataPosterOpts) (*DataPoster, error) {
	cfg := opts.Config()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	useNoOpStorage := cfg.UseNoOpStorage
	if opts.HeaderReader.IsParentChainArbitrum() && !cfg.UseNoOpStorage {
		useNoOpStorage = true
		log.Info("Disabling data poster storage, as parent chain appears to be an Arbitrum chain without a mempool")
	}
	encF := func() storage.EncoderDecoderInterface {
		if opts.Config().LegacyStorageEncoding {
			return &storage.LegacyEncoderDecoder{}
		}
		return &storage.EncoderDecoder{}
	}
	var queue state.QueueStorage
	switch {
	case useNoOpStorage:
		queue = &noop.Storage{}
	case opts.RedisClient != nil:
		var err error
		queue, err = redisstorage.NewStorage(opts.RedisClient, opts.RedisKey, &cfg.RedisSigner, encF)
		if err != nil {
			return nil, err
		}
	case cfg.UseDBStorage:
		storage := dbstorage.New(opts.Database, func() storage.EncoderDecoderInterface { return &storage.EncoderDecoder{} })
		if cfg.Dangerous.ClearDBStorage {
			if err := storage.PruneAll(ctx); err != nil {
				return nil, err
			}
		}
		queue = storage
	default:
		queue = slice.NewStorage(func() storage.EncoderDecoderInterface { return &storage.EncoderDecoder{} })
	}
	expression, err := govaluate.NewEvaluableExpression(cfg.MaxFeeCapFormula)
	if err != nil {
		return nil, fmt.Errorf("error creating govaluate evaluable expression for calculating maxFeeCap: %w", err)
	}
	dp := &DataPoster{
		headerReader: opts.HeaderReader,
		client:       opts.HeaderReader.Client(),
		auth:         opts.Auth,
		signer: func(_ context.Context, addr common.Address, tx *types.Transaction) (*types.Transaction, error) {
			return opts.Auth.Signer(addr, tx)
		},
		config:              opts.Config,
		usingNoOpStorage:    useNoOpStorage,
		metadataRetriever:   opts.MetadataRetriever,
		internalState:       state.NewInternalState(queue),
		maxFeeCapExpression: expression,
		extraBacklog:        opts.ExtraBacklog,
		parentChain:         opts.ParentChain,
	}
	var overflow bool
	dp.parentChainID256, overflow = uint256.FromBig(opts.ParentChain.ChainID)
	if overflow {
		return nil, fmt.Errorf("parent chain ID %v overflows uint256 (necessary for blob transactions)", opts.ParentChain.ChainID)
	}
	if dp.extraBacklog == nil {
		dp.extraBacklog = func() uint64 { return 0 }
	}
	if cfg.ExternalSigner.URL != "" {
		xsign, err := externalsigner.NewExternalSigner(ctx, &cfg.ExternalSigner)
		if err != nil {
			return nil, err
		}
		dp.signer = xsign.Signer
		dp.auth = xsign.TxOpts()
	}

	return dp, nil
}

func (p *DataPoster) Client() *ethclient.Client {
	return p.client
}

func (p *DataPoster) Config() *config.DataPosterConfig {
	return p.config()
}

func (p *DataPoster) ExtraBacklog() uint64 {
	return p.extraBacklog()
}

func (p *DataPoster) MaxFeeCapExpression() *govaluate.EvaluableExpression {
	return p.maxFeeCapExpression
}
func (p *DataPoster) ParentChain() *parent.ParentChain {
	return p.parentChain
}

func (p *DataPoster) Auth() *bind.TransactOpts {
	return p.auth
}

func (p *DataPoster) Sender() common.Address {
	return p.auth.From
}

func (p *DataPoster) MaxMempoolTransactions() uint64 {
	if p.usingNoOpStorage {
		return 1
	}
	config := p.config()
	return arbmath.MinInt(config.MaxMempoolTransactions, config.MaxMempoolWeight)
}

func (p *DataPoster) UsingNoOpStorage() bool {
	return p.usingNoOpStorage
}

func (p *DataPoster) HeaderReader() *headerreader.HeaderReader {
	return p.headerReader
}

func (p *DataPoster) InternalState() *state.InternalState {
	return p.internalState
}

func (p *DataPoster) RetrieveMetadata(ctx context.Context, blockNum *big.Int) ([]byte, error) {
	return p.metadataRetriever(ctx, blockNum)
}

func (p *DataPoster) PostSimpleTransaction(ctx context.Context, to common.Address, calldata []byte, gasLimit uint64, value *big.Int) (*types.Transaction, error) {
	lockedState := p.internalState.Lock()
	defer p.internalState.Unlock()
	nonce, err := lifecycle.GetNextNonceAndMaybeMeta(ctx, p, lockedState, 1)
	if err != nil {
		return nil, err
	}
	return p.postTransaction(ctx, lockedState, time.Now(), nonce.Nonce, nil, to, calldata, gasLimit, value, nil, nil)
}

func (p *DataPoster) PostTransaction(ctx context.Context, dataCreatedAt time.Time, nonce uint64, meta []byte, to common.Address, calldata []byte, gasLimit uint64, value *big.Int, kzgBlobs []kzg4844.Blob, accessList types.AccessList) (*types.Transaction, error) {
	lockedState := p.internalState.Lock()
	defer p.internalState.Unlock()
	return p.postTransaction(ctx, lockedState, dataCreatedAt, nonce, meta, to, calldata, gasLimit, value, kzgBlobs, accessList)
}

func (p *DataPoster) postTransaction(ctx context.Context, s *state.LockedInternalState, dataCreatedAt time.Time, nonce uint64, meta []byte, to common.Address, calldata []byte, gasLimit uint64, value *big.Int, kzgBlobs []kzg4844.Blob, accessList types.AccessList) (*types.Transaction, error) {
	if p.config().DisableNewTx {
		return nil, fmt.Errorf("posting new transaction is disabled")
	}

	var weight uint64 = 1
	if len(kzgBlobs) > 0 {
		weight = uint64(len(kzgBlobs))
	}
	expectedNonce, err := lifecycle.GetNextNonceAndMaybeMeta(ctx, p, s, weight)
	if err != nil {
		return nil, err
	}
	if nonce != expectedNonce.Nonce {
		return nil, fmt.Errorf("%w: data poster expected next transaction to have nonce %v but was requested to post transaction with nonce %v", storage.ErrStorageRace, expectedNonce.Nonce, nonce)
	}

	err = lifecycle.UpdateBalance(ctx, p, s)
	if err != nil {
		return nil, fmt.Errorf("failed to update data poster balance: %w", err)
	}

	latestHeader, err := p.headerReader.LastHeader(ctx)
	if err != nil {
		return nil, err
	}

	caps, err := fees.FeeAndTipCaps(ctx, p, s, nonce, gasLimit, uint64(len(kzgBlobs)), nil, dataCreatedAt, 0, latestHeader)
	if err != nil {
		return nil, err
	}

	var deprecatedData types.DynamicFeeTx
	var inner types.TxData
	replacementTimes := p.config().ReplacementTimes
	if len(kzgBlobs) > 0 {
		replacementTimes = p.config().BlobTxReplacementTimes
		value256, overflow := uint256.FromBig(value)
		if overflow {
			return nil, fmt.Errorf("blob transaction callvalue %v overflows uint256", value)
		}
		// Intentionally break out of date data poster redis clients,
		// so they don't try to replace by fee a tx they don't understand
		deprecatedData.Nonce = ^uint64(0)
		commitments, blobHashes, err := blobs.ComputeCommitmentsAndHashes(kzgBlobs)
		if err != nil {
			return nil, fmt.Errorf("failed to compute KZG commitments: %w", err)
		}
		proofs, version, err := blobs.ComputeProofs(kzgBlobs, commitments)
		if err != nil {
			return nil, fmt.Errorf("failed to compute KZG proofs: %w", err)
		}
		inner = &types.BlobTx{
			Nonce: nonce,
			Gas:   gasLimit,
			To:    to,
			Value: value256,
			Data:  calldata,
			Sidecar: &types.BlobTxSidecar{
				Version:     version,
				Blobs:       kzgBlobs,
				Commitments: commitments,
				Proofs:      proofs,
			},
			BlobHashes: blobHashes,
			AccessList: accessList,
			ChainID:    p.parentChainID256,
		}
		// reuse the code to convert gas fee and tip caps to uint256s
		err = fees.UpdateTxDataGasCaps(inner, caps.Fee.NonBlob, caps.Tip, caps.Fee.Blob)
		if err != nil {
			return nil, err
		}
	} else {
		deprecatedData = types.DynamicFeeTx{
			Nonce:      nonce,
			GasFeeCap:  caps.Fee.NonBlob,
			GasTipCap:  caps.Tip,
			Gas:        gasLimit,
			To:         &to,
			Value:      value,
			Data:       calldata,
			AccessList: accessList,
			ChainID:    p.parentChain.ChainID,
		}
		inner = &deprecatedData
	}
	fullTx, err := p.signer(ctx, p.Sender(), types.NewTx(inner))
	if err != nil {
		return nil, fmt.Errorf("signing transaction: %w", err)
	}
	cumulativeWeight := expectedNonce.CumulativeWeight + weight
	queuedTx := storage.QueuedTransaction{
		DeprecatedData:         deprecatedData,
		FullTx:                 fullTx,
		Meta:                   meta,
		Sent:                   false,
		Created:                dataCreatedAt,
		NextReplacement:        time.Now().Add(replacementTimes[0]),
		StoredCumulativeWeight: &cumulativeWeight,
	}
	return fullTx, p.sendTx(ctx, s, nil, &queuedTx)
}

func (p *DataPoster) saveTx(ctx context.Context, s *state.LockedInternalState, prevTx, newTx *storage.QueuedTransaction) error {
	if prevTx != nil {
		if prevTx.FullTx.Nonce() != newTx.FullTx.Nonce() {
			return fmt.Errorf("prevTx nonce %v doesn't match newTx nonce %v", prevTx.FullTx.Nonce(), newTx.FullTx.Nonce())
		}

		// Check if prevTx is the same as newTx and we don't need to do anything
		oldEnc, err := rlp.EncodeToBytes(prevTx)
		if err != nil {
			return fmt.Errorf("failed to encode prevTx: %w", err)
		}
		newEnc, err := rlp.EncodeToBytes(newTx)
		if err != nil {
			return fmt.Errorf("failed to encode newTx: %w", err)
		}
		if bytes.Equal(oldEnc, newEnc) {
			// No need to save newTx as it's the same as prevTx
			return nil
		}
	}
	if err := s.Queue.Put(ctx, newTx.FullTx.Nonce(), prevTx, newTx); err != nil {
		return fmt.Errorf("putting new tx in the queue: %w", err)
	}
	return nil
}

func (p *DataPoster) sendTx(ctx context.Context, s *state.LockedInternalState, prevTx *storage.QueuedTransaction, newTx *storage.QueuedTransaction) error {
	latestHeader, err := p.client.HeaderByNumber(ctx, nil)
	if err != nil {
		return err
	}
	var currentBlobFee *big.Int
	if p.config().Post4844Blobs && latestHeader.ExcessBlobGas != nil && latestHeader.BlobGasUsed != nil {
		currentBlobFee, err = p.parentChain.BlobFeePerByte(ctx, latestHeader)
		if err != nil {
			return err
		}
	}

	if arbmath.BigLessThan(newTx.FullTx.GasFeeCap(), latestHeader.BaseFee) {
		log.Info(
			"submitting transaction with GasFeeCap less than latest basefee",
			"txBasefeeCap", newTx.FullTx.GasFeeCap(),
			"latestBasefee", latestHeader.BaseFee,
			"elapsed", time.Since(newTx.Created),
		)
	}

	if newTx.FullTx.BlobGasFeeCap() != nil && currentBlobFee != nil && arbmath.BigLessThan(newTx.FullTx.BlobGasFeeCap(), currentBlobFee) {
		log.Info(
			"submitting transaction with BlobGasFeeCap less than latest blobfee",
			"txBlobGasFeeCap", newTx.FullTx.BlobGasFeeCap(),
			"latestBlobFee", currentBlobFee,
			"elapsed", time.Since(newTx.Created),
		)
	}

	if err := p.saveTx(ctx, s, prevTx, newTx); err != nil {
		return err
	}

	// The following check is to avoid sending transactions of a different type (e.g. DynamicFeeTxType vs BlobTxType)
	// to the previous tx if the previous tx is not yet included in a reorg resistant block, in order to avoid issues
	// where eventual consistency of parent chain mempools causes a tx with higher nonce blocking a tx of a
	// different type with a lower nonce.
	// If we decide not to send this tx yet, just leave it queued and with Sent set to false.
	// The resending/repricing loop in DataPoster.Start will keep trying.
	previouslySent := newTx.Sent || (prevTx != nil && prevTx.Sent) // if we've previously sent this nonce
	if !previouslySent && newTx.FullTx.Nonce() > 0 {
		precedingTx, err := s.Queue.Get(ctx, arbmath.SaturatingUSub(newTx.FullTx.Nonce(), 1))
		if err != nil {
			return fmt.Errorf("couldn't get preceding tx in DataPoster to check if should send tx with nonce %d: %w", newTx.FullTx.Nonce(), err)
		}
		if precedingTx != nil { // precedingTx == nil -> the actual preceding tx was already confirmed
			var latestBlockNumber, prevBlockNumber, reorgResistantTxCount uint64
			if precedingTx.FullTx.Type() != newTx.FullTx.Type() || !precedingTx.Sent {
				latestBlockNumber, err = p.client.BlockNumber(ctx)
				if err != nil {
					return fmt.Errorf("couldn't get block number in DataPoster to check if should send tx with nonce %d: %w", newTx.FullTx.Nonce(), err)
				}
				prevBlockNumber = arbmath.SaturatingUSub(latestBlockNumber, 1)
				reorgResistantTxCount, err = p.client.NonceAt(ctx, p.Sender(), new(big.Int).SetUint64(prevBlockNumber))
				if err != nil {
					return fmt.Errorf("couldn't determine reorg resistant nonce in DataPoster to check if should send tx with nonce %d: %w", newTx.FullTx.Nonce(), err)
				}

				if newTx.FullTx.Nonce() > reorgResistantTxCount {
					log.Info("DataPoster is avoiding creating a mempool nonce gap (the tx remains queued and will be retried)", "nonce", newTx.FullTx.Nonce(), "prevType", precedingTx.FullTx.Type(), "type", newTx.FullTx.Type(), "prevSent", precedingTx.Sent, "latestBlockNumber", latestBlockNumber, "prevBlockNumber", prevBlockNumber, "reorgResistantTxCount", reorgResistantTxCount)
					return nil
				}
			}
			log.Debug("DataPoster will send previously unsent batch tx", "nonce", newTx.FullTx.Nonce(), "prevType", precedingTx.FullTx.Type(), "type", newTx.FullTx.Type(), "prevSent", precedingTx.Sent, "latestBlockNumber", latestBlockNumber, "prevBlockNumber", prevBlockNumber, "reorgResistantTxCount", reorgResistantTxCount)
		}
	}

	if err := p.client.SendTransaction(ctx, newTx.FullTx); err != nil {
		isAlreadyKnown := rpcclient.IsAlreadyKnownError(err)
		isAlreadyKnown = isAlreadyKnown || strings.Contains(err.Error(), "nonce too low")
		// If we previously sent this nonce and the same tx, some L1 clients may return ReplacementNotAllowed instead of
		// an already known error (might be due to their cache size constraints) so we dont return an error in such a case
		_, _, errTxByHash := p.client.TransactionByHash(ctx, newTx.FullTx.Hash())
		isAlreadyKnown = isAlreadyKnown || (strings.Contains(err.Error(), "ReplacementNotAllowed") && errTxByHash == nil)
		if !isAlreadyKnown {
			log.Warn("DataPoster failed to send transaction", "err", err, "nonce", newTx.FullTx.Nonce(), "feeCap", newTx.FullTx.GasFeeCap(), "tipCap", newTx.FullTx.GasTipCap(), "blobFeeCap", newTx.FullTx.BlobGasFeeCap(), "gas", newTx.FullTx.Gas())
			return err
		}
		log.Info("DataPoster transaction already known", "err", err, "nonce", newTx.FullTx.Nonce(), "hash", newTx.FullTx.Hash())
	} else {
		log.Info("DataPoster sent transaction", "nonce", newTx.FullTx.Nonce(), "hash", newTx.FullTx.Hash(), "feeCap", newTx.FullTx.GasFeeCap(), "tipCap", newTx.FullTx.GasTipCap(), "blobFeeCap", newTx.FullTx.BlobGasFeeCap(), "gas", newTx.FullTx.Gas())
	}
	newerTx := *newTx
	newerTx.Sent = true
	return p.saveTx(ctx, s, newTx, &newerTx)
}

func (p *DataPoster) replaceTx(ctx context.Context, s *state.LockedInternalState, prevTx *storage.QueuedTransaction, backlogWeight uint64) error {
	latestHeader, err := p.headerReader.LastHeader(ctx)
	if err != nil {
		return err
	}

	caps, err := fees.FeeAndTipCaps(ctx, p, s, prevTx.FullTx.Nonce(), prevTx.FullTx.Gas(), uint64(len(prevTx.FullTx.BlobHashes())), prevTx.FullTx, prevTx.Created, backlogWeight, latestHeader)
	if err != nil {
		return err
	}

	minRbfIncrease := fees.MinRbfIncrease.SelectIfBlobs(len(prevTx.FullTx.BlobHashes()) > 0)
	newTx := *prevTx
	if (prevTx.FullTx.GasFeeCap().Sign() > 0 && arbmath.BigDivToBips(caps.Fee.NonBlob, prevTx.FullTx.GasFeeCap()) < minRbfIncrease) ||
		(prevTx.FullTx.BlobGasFeeCap() != nil && prevTx.FullTx.BlobGasFeeCap().Sign() > 0 && arbmath.BigDivToBips(caps.Fee.Blob, prevTx.FullTx.BlobGasFeeCap()) < minRbfIncrease) {
		log.Debug(
			"no need to replace by fee transaction",
			"nonce", prevTx.FullTx.Nonce(),
			"lastFeeCap", prevTx.FullTx.GasFeeCap(),
			"recommendedFeeCap", caps.Fee.NonBlob,
			"lastTipCap", prevTx.FullTx.GasTipCap(),
			"recommendedTipCap", caps.Tip,
			"lastBlobFeeCap", prevTx.FullTx.BlobGasFeeCap(),
			"recommendedBlobFeeCap", caps.Fee.Blob,
		)
		newTx.NextReplacement = time.Now().Add(time.Minute)
		return p.sendTx(ctx, s, prevTx, &newTx)
	}

	replacementTimes := p.config().ReplacementTimes
	if len(prevTx.FullTx.BlobHashes()) > 0 {
		replacementTimes = p.config().BlobTxReplacementTimes
	}

	elapsed := time.Since(prevTx.Created)
	for _, replacement := range replacementTimes {
		if elapsed >= replacement {
			continue
		}
		newTx.NextReplacement = prevTx.Created.Add(replacement)
		break
	}
	newTx.Sent = false
	newTx.DeprecatedData.GasFeeCap = caps.Fee.NonBlob
	newTx.DeprecatedData.GasTipCap = caps.Tip
	unsignedTx, err := fees.UpdateGasCaps(newTx.FullTx, caps.Fee.NonBlob, caps.Tip, caps.Fee.Blob)
	if err != nil {
		return err
	}
	newTx.FullTx, err = p.signer(ctx, p.Sender(), unsignedTx)
	if err != nil {
		return err
	}

	return p.sendTx(ctx, s, prevTx, &newTx)
}

const minWait = time.Second * 10

// Tries to acquire redis lock, updates balance and nonce,
func (p *DataPoster) Start(ctxIn context.Context) {
	p.StopWaiter.Start(ctxIn, p)
	p.CallIteratively(func(ctx context.Context) time.Duration {
		lockedState := p.internalState.Lock()
		defer p.internalState.Unlock()
		err := lifecycle.UpdateBalance(ctx, p, lockedState)
		if err != nil {
			log.Warn("failed to update tx poster balance", "err", err)
			return minWait
		}
		err = lifecycle.UpdateNonce(ctx, p, lockedState)
		if err != nil {
			// This is non-fatal because it's only needed for clearing out old queue items.
			log.Warn("failed to update tx poster nonce", "err", err)
		}
		now := time.Now()
		nextCheck := now.Add(p.config().ReplacementTimes[0])
		if len(p.config().BlobTxReplacementTimes) > 0 {
			nextCheck = now.Add(arbmath.MinInt(p.config().ReplacementTimes[0], p.config().BlobTxReplacementTimes[0]))
		}
		maxTxsToRbf := p.config().MaxMempoolTransactions
		if maxTxsToRbf == 0 {
			maxTxsToRbf = 512
		}
		unconfirmedNonce, err := p.client.NonceAt(ctx, p.Sender(), nil)
		if err != nil {
			log.Warn("Failed to get latest nonce", "err", err)
			return minWait
		}
		// #nosec G115
		datapostermetrics.LatestUnconfirmedNonceGauge.Update(int64(unconfirmedNonce))
		// We use unconfirmedNonce here to replace-by-fee transactions that aren't in a block,
		// excluding those that are in an unconfirmed block. If a reorg occurs, we'll continue
		// replacing them by fee.
		queueContents, err := lockedState.Queue.FetchContents(ctx, unconfirmedNonce, maxTxsToRbf)
		if err != nil {
			log.Error("Failed to fetch tx queue contents", "err", err)
			return minWait
		}
		latestQueued, err := lockedState.Queue.FetchLast(ctx)
		if err != nil {
			log.Error("Failed to fetch last queued tx", "err", err)
			return minWait
		}
		var latestCumulativeWeight, latestNonce uint64
		if latestQueued != nil {
			latestCumulativeWeight = latestQueued.CumulativeWeight()
			latestNonce = latestQueued.FullTx.Nonce()

			confirmedNonce := unconfirmedNonce - 1
			confirmedMeta, err := lockedState.Queue.Get(ctx, confirmedNonce)
			if err == nil && confirmedMeta != nil {
				// #nosec G115
				datapostermetrics.TotalQueueWeightGauge.Update(int64(arbmath.SaturatingUSub(latestCumulativeWeight, confirmedMeta.CumulativeWeight())))
				// #nosec G115
				datapostermetrics.TotalQueueLengthGauge.Update(int64(arbmath.SaturatingUSub(latestNonce, confirmedNonce)))
			} else {
				log.Error("Failed to fetch latest confirmed tx from queue", "confirmedNonce", confirmedNonce, "err", err, "confirmedMeta", confirmedMeta)
			}
		}

		for _, tx := range queueContents {
			if now.After(tx.NextReplacement) {
				weightBacklog := arbmath.SaturatingUSub(latestCumulativeWeight, tx.CumulativeWeight())
				nonceBacklog := arbmath.SaturatingUSub(latestNonce, tx.FullTx.Nonce())
				err := p.replaceTx(ctx, lockedState, tx, arbmath.MaxInt(nonceBacklog, weightBacklog))
				lifecycle.MaybeLogError(err, lockedState, tx, "failed to replace-by-fee transaction")
			} else {
				err := p.sendTx(ctx, lockedState, tx, tx)
				lifecycle.MaybeLogError(err, lockedState, tx, "failed to re-send transaction")
			}
			nonce := tx.FullTx.Nonce()
			tx, err = lockedState.Queue.Get(ctx, nonce)
			if err != nil {
				log.Error("Failed to fetch tx from queue to check updated status", "nonce", nonce, "err", err)
				return minWait
			}
			if tx == nil {
				log.Error("Failed to fetch tx from queue to check updated status, got tx == nil", "nonce", nonce)
				return minWait
			}
			if nextCheck.After(tx.NextReplacement) {
				nextCheck = tx.NextReplacement
			}
			if !tx.Sent {
				// We can't progress any further if we failed to send this tx
				// Retry sending this tx soon
				return minWait
			}
		}
		wait := time.Until(nextCheck)
		if wait < minWait {
			wait = minWait
		}
		return wait
	})
}
