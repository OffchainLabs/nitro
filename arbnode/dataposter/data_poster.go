// Copyright 2021-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

// Package dataposter implements generic functionality to post transactions.
package dataposter

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"math"
	"math/big"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Knetic/govaluate"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
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
// persistent storage is used for the queue, after restarting the node
// dataposter will pick up where it left.
// DataPoster must be RLP serializable and deserializable
type DataPoster struct {
	stopwaiter.StopWaiter
	headerReader      *headerreader.HeaderReader
	client            arbutil.L1Interface
	auth              *bind.TransactOpts
	signer            signerFn
	redisLock         AttemptLocker
	config            ConfigFetcher
	usingNoOpStorage  bool
	replacementTimes  []time.Duration
	metadataRetriever func(ctx context.Context, blockNum *big.Int) ([]byte, error)
	extraBacklog      func() uint64

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

	maxFeeCapExpression *govaluate.EvaluableExpression
}

// signerFn is a signer function callback when a contract requires a method to
// sign the transaction before submission.
// This can be local or external, hence the context parameter.
type signerFn func(context.Context, common.Address, *types.Transaction) (*types.Transaction, error)

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
		log.Warn("Disabling replace-by-fee for data poster")
	}
	// To avoid special casing "don't replace again", replace in 10 years.
	return append(res, time.Hour*24*365*10), nil
}

type DataPosterOpts struct {
	Database          ethdb.Database
	HeaderReader      *headerreader.HeaderReader
	Auth              *bind.TransactOpts
	RedisClient       redis.UniversalClient
	RedisLock         AttemptLocker
	Config            ConfigFetcher
	MetadataRetriever func(ctx context.Context, blockNum *big.Int) ([]byte, error)
	ExtraBacklog      func() uint64
	RedisKey          string // Redis storage key
}

func NewDataPoster(ctx context.Context, opts *DataPosterOpts) (*DataPoster, error) {
	cfg := opts.Config()
	replacementTimes, err := parseReplacementTimes(cfg.ReplacementTimes)
	if err != nil {
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
	var queue QueueStorage
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
		replacementTimes:    replacementTimes,
		metadataRetriever:   opts.MetadataRetriever,
		queue:               queue,
		redisLock:           opts.RedisLock,
		errorCount:          make(map[uint64]int),
		maxFeeCapExpression: expression,
		extraBacklog:        opts.ExtraBacklog,
	}
	if dp.extraBacklog == nil {
		dp.extraBacklog = func() uint64 { return 0 }
	}
	if cfg.ExternalSigner.URL != "" {
		signer, sender, err := externalSigner(ctx, &cfg.ExternalSigner)
		if err != nil {
			return nil, err
		}
		dp.signer = signer
		dp.auth = &bind.TransactOpts{
			From: sender,
			Signer: func(address common.Address, tx *types.Transaction) (*types.Transaction, error) {
				return signer(context.TODO(), address, tx)
			},
		}
	}
	return dp, nil
}

func rpcClient(ctx context.Context, opts *ExternalSignerCfg) (*rpc.Client, error) {
	tlsCfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	if opts.ClientCert != "" && opts.ClientPrivateKey != "" {
		log.Info("Client certificate for external signer is enabled")
		clientCert, err := tls.LoadX509KeyPair(opts.ClientCert, opts.ClientPrivateKey)
		if err != nil {
			return nil, fmt.Errorf("error loading client certificate and private key: %w", err)
		}
		tlsCfg.Certificates = []tls.Certificate{clientCert}
	}

	if opts.RootCA != "" {
		rootCrt, err := os.ReadFile(opts.RootCA)
		if err != nil {
			return nil, fmt.Errorf("error reading external signer root CA: %w", err)
		}
		rootCertPool := x509.NewCertPool()
		rootCertPool.AppendCertsFromPEM(rootCrt)
		tlsCfg.RootCAs = rootCertPool
	}

	return rpc.DialOptions(
		ctx,
		opts.URL,
		rpc.WithHTTPClient(
			&http.Client{
				Transport: &http.Transport{
					TLSClientConfig: tlsCfg,
				},
			},
		),
	)
}

// txToSendTxArgs converts transaction to SendTxArgs. This is needed for
// external signer to specify From field.
func txToSendTxArgs(addr common.Address, tx *types.Transaction) (*apitypes.SendTxArgs, error) {
	if tx.To() == nil {
		return nil, fmt.Errorf("transaction %v has no destination", tx.Hash())
	}
	to := common.NewMixedcaseAddress(*tx.To())
	data := (hexutil.Bytes)(tx.Data())
	val := (*hexutil.Big)(tx.Value())
	if val == nil {
		val = (*hexutil.Big)(big.NewInt(0))
	}
	al := tx.AccessList()
	return &apitypes.SendTxArgs{
		From:                 common.NewMixedcaseAddress(addr),
		To:                   &to,
		Gas:                  hexutil.Uint64(tx.Gas()),
		GasPrice:             (*hexutil.Big)(tx.GasPrice()),
		MaxFeePerGas:         (*hexutil.Big)(tx.GasFeeCap()),
		MaxPriorityFeePerGas: (*hexutil.Big)(tx.GasTipCap()),
		Value:                *val,
		Nonce:                hexutil.Uint64(tx.Nonce()),
		Data:                 &data,
		AccessList:           &al,
		ChainID:              (*hexutil.Big)(tx.ChainId()),
	}, nil
}

// externalSigner returns signer function and ethereum address of the signer.
// Returns an error if address isn't specified or if it can't connect to the
// signer RPC server.
func externalSigner(ctx context.Context, opts *ExternalSignerCfg) (signerFn, common.Address, error) {
	if opts.Address == "" {
		return nil, common.Address{}, errors.New("external signer (From) address specified")
	}

	client, err := rpcClient(ctx, opts)
	if err != nil {
		return nil, common.Address{}, fmt.Errorf("error connecting external signer: %w", err)
	}
	sender := common.HexToAddress(opts.Address)
	return func(ctx context.Context, addr common.Address, tx *types.Transaction) (*types.Transaction, error) {
		// According to the "eth_signTransaction" API definition, this should be
		// RLP encoded transaction object.
		// https://ethereum.org/en/developers/docs/apis/json-rpc/#eth_signtransaction
		var data hexutil.Bytes
		args, err := txToSendTxArgs(addr, tx)
		if err != nil {
			return nil, fmt.Errorf("error converting transaction to sendTxArgs: %w", err)
		}
		if err := client.CallContext(ctx, &data, opts.Method, args); err != nil {
			return nil, fmt.Errorf("signing transaction: %w", err)
		}
		var signedTx types.Transaction
		if err := rlp.DecodeBytes(data, &signedTx); err != nil {
			return nil, fmt.Errorf("error decoding signed transaction: %w", err)
		}
		hasher := types.LatestSignerForChainID(tx.ChainId())
		if h := hasher.Hash(args.ToTransaction()); h != hasher.Hash(&signedTx) {
			return nil, fmt.Errorf("transaction: %x from external signer differs from request: %x", hasher.Hash(&signedTx), h)
		}
		return &signedTx, nil
	}, sender, nil
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
	return p.config().MaxMempoolTransactions
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
		unconfirmedNonce, err := p.client.NonceAt(ctx, p.Sender(), nil)
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
		nonce, err := p.client.NonceAt(ctx, p.Sender(), nonceQueryBlock)
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

// evalMaxFeeCapExpr uses MaxFeeCapFormula from config to calculate the expression's result by plugging in appropriate parameter values
// backlogOfBatches should already include extraBacklog
func (p *DataPoster) evalMaxFeeCapExpr(backlogOfBatches uint64, elapsed time.Duration) (*big.Int, error) {
	config := p.config()
	parameters := map[string]any{
		"BacklogOfBatches":      float64(backlogOfBatches),
		"UrgencyGWei":           config.UrgencyGwei,
		"ElapsedTime":           float64(elapsed),
		"ElapsedTimeBase":       float64(config.ElapsedTimeBase),
		"ElapsedTimeImportance": config.ElapsedTimeImportance,
		"TargetPriceGWei":       config.TargetPriceGwei,
	}
	result, err := p.maxFeeCapExpression.Evaluate(parameters)
	if err != nil {
		return nil, fmt.Errorf("error evaluating maxFeeCapExpression: %w", err)
	}
	resultFloat, ok := result.(float64)
	if !ok {
		// This shouldn't be possible because we only pass in float64s as arguments
		return nil, fmt.Errorf("maxFeeCapExpression evaluated to non-float64: %v", result)
	}
	// 1e9 gwei gas price is practically speaking an infinite gas price, so we cap it there.
	// This also allows the formula to return positive infinity safely.
	resultFloat = math.Min(resultFloat, 1e9)
	resultBig := arbmath.FloatToBig(resultFloat * params.GWei)
	if resultBig == nil {
		return nil, fmt.Errorf("maxFeeCapExpression evaluated to float64 not convertible to integer: %v", resultFloat)
	}
	if resultBig.Sign() < 0 {
		return nil, fmt.Errorf("maxFeeCapExpression evaluated < 0: %v", resultFloat)
	}
	return resultBig, nil
}

var big4 = big.NewInt(4)

// The dataPosterBacklog argument should *not* include extraBacklog (it's added in in this function)
func (p *DataPoster) feeAndTipCaps(ctx context.Context, nonce uint64, gasLimit uint64, lastFeeCap *big.Int, lastTipCap *big.Int, dataCreatedAt time.Time, dataPosterBacklog uint64) (*big.Int, *big.Int, error) {
	config := p.config()
	dataPosterBacklog += p.extraBacklog()
	latestHeader, err := p.headerReader.LastHeader(ctx)
	if err != nil {
		return nil, nil, err
	}
	if latestHeader.BaseFee == nil {
		return nil, nil, fmt.Errorf("latest parent chain block %v missing BaseFee (either the parent chain does not have EIP-1559 or the parent chain node is not synced)", latestHeader.Number)
	}
	softConfBlock := arbmath.BigSubByUint(latestHeader.Number, config.NonceRbfSoftConfs)
	softConfNonce, err := p.client.NonceAt(ctx, p.Sender(), softConfBlock)
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
	maxFeeCap, err := p.evalMaxFeeCapExpr(dataPosterBacklog, elapsed)
	if err != nil {
		return nil, nil, err
	}
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
	if config.AllocateMempoolBalance && !p.usingNoOpStorage {
		// We split the transactions into three groups:
		// - The first transaction gets 1/2 of the balance.
		// - The first half of transactions get 1/3 of the balance split among them.
		// - The remaining transactions get the remaining 1/6 of the balance split among them.
		// This helps ensure batch posting is reliable under a variety of fee conditions.
		// With noop storage, we don't try to replace-by-fee, so we don't need to worry about this.
		balanceForTx.Div(balanceForTx, common.Big2)
		if nonce != softConfNonce && config.MaxMempoolTransactions > 1 {
			// Compared to dividing the remaining transactions by balance equally,
			// the first half of transactions should get a 4/3 weight,
			// and the remaining half should get a 2/3 weight.
			// This makes sure the average weight is 1, and the first half of transactions
			// have twice the weight of the second half of transactions.
			// The +1 and -1 here are to account for the first transaction being handled separately.
			if nonce > softConfNonce && nonce < softConfNonce+1+(config.MaxMempoolTransactions-1)/2 {
				balanceForTx.Mul(balanceForTx, big4)
			} else {
				balanceForTx.Mul(balanceForTx, common.Big2)
			}
			balanceForTx.Div(balanceForTx, common.Big3)
			// After weighting, split the balance between each of the transactions
			// other than the first tx which already got half.
			// balanceForTx /= config.MaxMempoolTransactions-1
			balanceForTx.Div(balanceForTx, arbmath.UintToBig(config.MaxMempoolTransactions-1))
		}
	}
	balanceFeeCap := arbmath.BigDivByUint(balanceForTx, gasLimit)
	if arbmath.BigGreaterThan(newFeeCap, balanceFeeCap) {
		log.Warn(
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

func (p *DataPoster) PostTransaction(ctx context.Context, dataCreatedAt time.Time, nonce uint64, meta []byte, to common.Address, calldata []byte, gasLimit uint64, value *big.Int, accessList types.AccessList) (*types.Transaction, error) {
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
		Nonce:      nonce,
		GasTipCap:  tipCap,
		GasFeeCap:  feeCap,
		Gas:        gasLimit,
		To:         &to,
		Value:      value,
		Data:       calldata,
		AccessList: accessList,
	}
	fullTx, err := p.signer(ctx, p.Sender(), types.NewTx(&inner))
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
	if prevTx != nil {
		if prevTx.Data.Nonce != newTx.Data.Nonce {
			return fmt.Errorf("prevTx nonce %v doesn't match newTx nonce %v", prevTx.Data.Nonce, newTx.Data.Nonce)
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
	if err := p.queue.Put(ctx, newTx.Data.Nonce, prevTx, newTx); err != nil {
		return fmt.Errorf("putting new tx in the queue: %w", err)
	}
	return nil
}

func (p *DataPoster) sendTx(ctx context.Context, prevTx *storage.QueuedTransaction, newTx *storage.QueuedTransaction) error {
	if err := p.saveTx(ctx, prevTx, newTx); err != nil {
		return err
	}
	if err := p.client.SendTransaction(ctx, newTx.FullTx); err != nil {
		if !strings.Contains(err.Error(), "already known") && !strings.Contains(err.Error(), "nonce too low") {
			log.Warn("DataPoster failed to send transaction", "err", err, "nonce", newTx.FullTx.Nonce(), "feeCap", newTx.FullTx.GasFeeCap(), "tipCap", newTx.FullTx.GasTipCap(), "gas", newTx.FullTx.Gas())
			return err
		}
		log.Info("DataPoster transaction already known", "err", err, "nonce", newTx.FullTx.Nonce(), "hash", newTx.FullTx.Hash())
	} else {
		log.Info("DataPoster sent transaction", "nonce", newTx.FullTx.Nonce(), "hash", newTx.FullTx.Hash(), "feeCap", newTx.FullTx.GasFeeCap(), "tipCap", newTx.FullTx.GasTipCap(), "gas", newTx.FullTx.Gas())
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
	newTx.FullTx, err = p.signer(ctx, p.Sender(), types.NewTx(&newTx.Data))
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
	nonce, err := p.client.NonceAt(ctx, p.Sender(), header.Number)
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
	balance, err := p.client.BalanceAt(ctx, p.Sender(), big.NewInt(-1))
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
		unconfirmedNonce, err := p.client.NonceAt(ctx, p.Sender(), nil)
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
	WaitForL1Finality      bool              `koanf:"wait-for-l1-finality" reload:"hot"`
	MaxMempoolTransactions uint64            `koanf:"max-mempool-transactions" reload:"hot"`
	MaxQueuedTransactions  int               `koanf:"max-queued-transactions" reload:"hot"`
	TargetPriceGwei        float64           `koanf:"target-price-gwei" reload:"hot"`
	UrgencyGwei            float64           `koanf:"urgency-gwei" reload:"hot"`
	MinFeeCapGwei          float64           `koanf:"min-fee-cap-gwei" reload:"hot"`
	MinTipCapGwei          float64           `koanf:"min-tip-cap-gwei" reload:"hot"`
	MaxTipCapGwei          float64           `koanf:"max-tip-cap-gwei" reload:"hot"`
	NonceRbfSoftConfs      uint64            `koanf:"nonce-rbf-soft-confs" reload:"hot"`
	AllocateMempoolBalance bool              `koanf:"allocate-mempool-balance" reload:"hot"`
	UseDBStorage           bool              `koanf:"use-db-storage"`
	UseNoOpStorage         bool              `koanf:"use-noop-storage"`
	LegacyStorageEncoding  bool              `koanf:"legacy-storage-encoding" reload:"hot"`
	Dangerous              DangerousConfig   `koanf:"dangerous"`
	ExternalSigner         ExternalSignerCfg `koanf:"external-signer"`
	MaxFeeCapFormula       string            `koanf:"max-fee-cap-formula" reload:"hot"`
	ElapsedTimeBase        time.Duration     `koanf:"elapsed-time-base" reload:"hot"`
	ElapsedTimeImportance  float64           `koanf:"elapsed-time-importance" reload:"hot"`
}

type ExternalSignerCfg struct {
	// URL of the external signer rpc server, if set this overrides transaction
	// options and uses external signer
	// for signing transactions.
	URL string `koanf:"url"`
	// Hex encoded ethereum address of the external signer.
	Address string `koanf:"address"`
	// API method name (e.g. eth_signTransaction).
	Method string `koanf:"method"`
	// (Optional) Path to the external signer root CA certificate.
	// This allows us to use self-signed certificats on the external signer.
	RootCA string `koanf:"root-ca"`
	// (Optional) Client certificate for mtls.
	ClientCert string `koanf:"client-cert"`
	// (Optional) Client certificate key for mtls.
	// This is required when client-cert is set.
	ClientPrivateKey string `koanf:"client-private-key"`
}

type DangerousConfig struct {
	// This should be used with caution, only when dataposter somehow gets in a
	// bad state and we require clearing it.
	ClearDBStorage bool `koanf:"clear-dbstorage"`
}

// ConfigFetcher function type is used instead of directly passing config so
// that flags can be reloaded dynamically.
type ConfigFetcher func() *DataPosterConfig

func DataPosterConfigAddOptions(prefix string, f *pflag.FlagSet, defaultDataPosterConfig DataPosterConfig) {
	f.String(prefix+".replacement-times", defaultDataPosterConfig.ReplacementTimes, "comma-separated list of durations since first posting to attempt a replace-by-fee")
	f.Bool(prefix+".wait-for-l1-finality", defaultDataPosterConfig.WaitForL1Finality, "only treat a transaction as confirmed after L1 finality has been achieved (recommended)")
	f.Uint64(prefix+".max-mempool-transactions", defaultDataPosterConfig.MaxMempoolTransactions, "the maximum number of transactions to have queued in the mempool at once (0 = unlimited)")
	f.Int(prefix+".max-queued-transactions", defaultDataPosterConfig.MaxQueuedTransactions, "the maximum number of unconfirmed transactions to track at once (0 = unlimited)")
	f.Float64(prefix+".target-price-gwei", defaultDataPosterConfig.TargetPriceGwei, "the target price to use for maximum fee cap calculation")
	f.Float64(prefix+".urgency-gwei", defaultDataPosterConfig.UrgencyGwei, "the urgency to use for maximum fee cap calculation")
	f.Float64(prefix+".min-fee-cap-gwei", defaultDataPosterConfig.MinFeeCapGwei, "the minimum fee cap to post transactions at")
	f.Float64(prefix+".min-tip-cap-gwei", defaultDataPosterConfig.MinTipCapGwei, "the minimum tip cap to post transactions at")
	f.Float64(prefix+".max-tip-cap-gwei", defaultDataPosterConfig.MaxTipCapGwei, "the maximum tip cap to post transactions at")
	f.Uint64(prefix+".nonce-rbf-soft-confs", defaultDataPosterConfig.NonceRbfSoftConfs, "the maximum probable reorg depth, used to determine when a transaction will no longer likely need replaced-by-fee")
	f.Bool(prefix+".allocate-mempool-balance", defaultDataPosterConfig.AllocateMempoolBalance, "if true, don't put transactions in the mempool that spend a total greater than the batch poster's balance")
	f.Bool(prefix+".use-db-storage", defaultDataPosterConfig.UseDBStorage, "uses database storage when enabled")
	f.Bool(prefix+".use-noop-storage", defaultDataPosterConfig.UseNoOpStorage, "uses noop storage, it doesn't store anything")
	f.Bool(prefix+".legacy-storage-encoding", defaultDataPosterConfig.LegacyStorageEncoding, "encodes items in a legacy way (as it was before dropping generics)")
	f.String(prefix+".max-fee-cap-formula", defaultDataPosterConfig.MaxFeeCapFormula, "mathematical formula to calculate maximum fee cap gwei the result of which would be float64.\n"+
		"This expression is expected to be evaluated please refer https://github.com/Knetic/govaluate/blob/master/MANUAL.md to find all available mathematical operators.\n"+
		"Currently available variables to construct the formula are BacklogOfBatches, UrgencyGWei, ElapsedTime, ElapsedTimeBase, ElapsedTimeImportance, and TargetPriceGWei")
	f.Duration(prefix+".elapsed-time-base", defaultDataPosterConfig.ElapsedTimeBase, "unit to measure the time elapsed since creation of transaction used for maximum fee cap calculation")
	f.Float64(prefix+".elapsed-time-importance", defaultDataPosterConfig.ElapsedTimeImportance, "weight given to the units of time elapsed used for maximum fee cap calculation")

	signature.SimpleHmacConfigAddOptions(prefix+".redis-signer", f)
	addDangerousOptions(prefix+".dangerous", f)
	addExternalSignerOptions(prefix+".external-signer", f)
}

func addDangerousOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".clear-dbstorage", DefaultDataPosterConfig.Dangerous.ClearDBStorage, "clear database storage")
}

func addExternalSignerOptions(prefix string, f *pflag.FlagSet) {
	f.String(prefix+".url", DefaultDataPosterConfig.ExternalSigner.URL, "external signer url")
	f.String(prefix+".address", DefaultDataPosterConfig.ExternalSigner.Address, "external signer address")
	f.String(prefix+".method", DefaultDataPosterConfig.ExternalSigner.Method, "external signer method")
	f.String(prefix+".root-ca", DefaultDataPosterConfig.ExternalSigner.RootCA, "external signer root CA")
	f.String(prefix+".client-cert", DefaultDataPosterConfig.ExternalSigner.ClientCert, "rpc client cert")
	f.String(prefix+".client-private-key", DefaultDataPosterConfig.ExternalSigner.ClientPrivateKey, "rpc client private key")
}

var DefaultDataPosterConfig = DataPosterConfig{
	ReplacementTimes:       "5m,10m,20m,30m,1h,2h,4h,6h,8h,12h,16h,18h,20h,22h",
	WaitForL1Finality:      true,
	TargetPriceGwei:        60.,
	UrgencyGwei:            2.,
	MaxMempoolTransactions: 20,
	MinTipCapGwei:          0.05,
	MaxTipCapGwei:          5,
	NonceRbfSoftConfs:      1,
	AllocateMempoolBalance: true,
	UseDBStorage:           true,
	UseNoOpStorage:         false,
	LegacyStorageEncoding:  false,
	Dangerous:              DangerousConfig{ClearDBStorage: false},
	ExternalSigner:         ExternalSignerCfg{Method: "eth_signTransaction"},
	MaxFeeCapFormula:       "((BacklogOfBatches * UrgencyGWei) ** 2) + ((ElapsedTime/ElapsedTimeBase) ** 2) * ElapsedTimeImportance + TargetPriceGWei",
	ElapsedTimeBase:        10 * time.Minute,
	ElapsedTimeImportance:  10,
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
	MaxMempoolTransactions: 20,
	MinTipCapGwei:          0.05,
	MaxTipCapGwei:          5,
	NonceRbfSoftConfs:      1,
	AllocateMempoolBalance: true,
	UseDBStorage:           false,
	UseNoOpStorage:         false,
	LegacyStorageEncoding:  false,
	ExternalSigner:         ExternalSignerCfg{Method: "eth_signTransaction"},
	MaxFeeCapFormula:       "((BacklogOfBatches * UrgencyGWei) ** 2) + ((ElapsedTime/ElapsedTimeBase) ** 2) * ElapsedTimeImportance + TargetPriceGWei",
	ElapsedTimeBase:        10 * time.Minute,
	ElapsedTimeImportance:  10,
}

var TestDataPosterConfigForValidator = func() DataPosterConfig {
	config := TestDataPosterConfig
	config.MaxMempoolTransactions = 1 // the validator cannot queue transactions
	return config
}()
