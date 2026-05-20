// Copyright 2023-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package dataposter

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/Knetic/govaluate"
	"github.com/google/go-cmp/cmp"
	"github.com/holiman/uint256"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/txpool/legacypool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/arbnode/dataposter/config"
	"github.com/offchainlabs/nitro/arbnode/dataposter/externalsigner"
	"github.com/offchainlabs/nitro/arbnode/dataposter/externalsignertest"
	"github.com/offchainlabs/nitro/arbnode/dataposter/fees"
	"github.com/offchainlabs/nitro/arbnode/dataposter/state"
	"github.com/offchainlabs/nitro/arbnode/dataposter/storage"
	"github.com/offchainlabs/nitro/arbnode/parent"
	"github.com/offchainlabs/nitro/util/arbmath"
)

var (
	blobTx = types.NewTx(
		&types.BlobTx{
			ChainID:   uint256.NewInt(1337),
			Nonce:     13,
			GasTipCap: uint256.NewInt(1),
			GasFeeCap: uint256.NewInt(1),
			Gas:       3,
			To:        common.Address{},
			Value:     uint256.NewInt(1),
			Data:      []byte{0x01, 0x02, 0x03},
			BlobHashes: []common.Hash{
				common.BigToHash(big.NewInt(1)),
				common.BigToHash(big.NewInt(2)),
				common.BigToHash(big.NewInt(3)),
			},
			Sidecar: &types.BlobTxSidecar{},
		},
	)
	dynamicFeeTx = types.NewTx(
		&types.DynamicFeeTx{
			Nonce:     13,
			GasTipCap: big.NewInt(1),
			GasFeeCap: big.NewInt(1),
			Gas:       3,
			To:        nil,
			Value:     big.NewInt(1),
			Data:      []byte{0x01, 0x02, 0x03},
		},
	)
)

func TestExternalSigner(t *testing.T) {
	srv := externalsignertest.NewServer(t)
	go func() {
		if err := srv.Start(); err != nil {
			log.Error("Failed to start external signer server:", err)
			return
		}
	}()
	signerCfg, err := config.ExternalSignerTestCfg(srv.Address, srv.URL())
	if err != nil {
		t.Fatalf("Error getting signer test config: %v", err)
	}
	ctx := context.Background()
	xsign, err := externalsigner.NewExternalSigner(ctx, signerCfg)
	if err != nil {
		t.Fatalf("Error getting external signer: %v", err)
	}

	for _, tc := range []struct {
		desc string
		tx   *types.Transaction
	}{
		{
			desc: "blob transaction",
			tx:   blobTx,
		},
		{
			desc: "dynamic fee transaction",
			tx:   dynamicFeeTx,
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			{
				got, err := xsign.Signer(ctx, xsign.Sender, tc.tx)
				if err != nil {
					t.Fatalf("Error signing transaction with external signer: %v", err)
				}
				want, err := srv.SignerFn(xsign.Sender, tc.tx)
				if err != nil {
					t.Fatalf("Error signing transaction: %v", err)
				}
				if diff := cmp.Diff(want.Hash(), got.Hash()); diff != "" {
					t.Errorf("Signing transaction: unexpected diff: %v\n", diff)
				}
				hasher := types.LatestSignerForChainID(tc.tx.ChainId())
				if h, g := hasher.Hash(tc.tx), hasher.Hash(got); h != g {
					t.Errorf("Signed transaction hash: %v differs from initial transaction hash: %v", g, h)
				}
			}
		})
	}
}

func TestMaxFeeCapFormulaCalculation(t *testing.T) {
	// This test alerts, by failing, if the max fee cap formula were to be changed in the DefaultDataPosterConfig to
	// use new variables other than the ones that are keys of 'parameters' map below
	expression, err := govaluate.NewEvaluableExpression(config.DefaultDataPosterConfig.MaxFeeCapFormula)
	if err != nil {
		t.Fatalf("Error creating govaluate evaluable expression for calculating default maxFeeCap formula: %v", err)
	}
	cfg := config.DefaultDataPosterConfig
	cfg.TargetPriceGwei = 0
	p := &DataPoster{
		config:              func() *config.DataPosterConfig { return &cfg },
		internalState:       state.NewInternalState(nil),
		maxFeeCapExpression: expression,
	}
	result, err := fees.EvalMaxFeeCapExpr(p, 0, 0)
	if err != nil {
		t.Fatalf("Error evaluating MaxFeeCap expression: %v", err)
	}
	if result.Cmp(common.Big0) != 0 {
		t.Fatalf("Unexpected result. Got: %d, want: 0", result)
	}

	result, err = fees.EvalMaxFeeCapExpr(p, 0, time.Since(time.Time{}))
	if err != nil {
		t.Fatalf("Error evaluating MaxFeeCap expression: %v", err)
	}
	if result.Cmp(big.NewInt(params.GWei)) <= 0 {
		t.Fatalf("Unexpected result. Got: %d, want: >0", result)
	}
}

type stubL1ClientInner struct {
	senderNonce        uint64
	suggestedGasTipCap *big.Int
}

func (c *stubL1ClientInner) CallContext(ctx_in context.Context, result interface{}, method string, args ...interface{}) error {
	switch method {
	case "eth_getTransactionCount":
		ptr, ok := result.(*hexutil.Uint64)
		if !ok {
			return errors.New("result is not a *hexutil.Uint64")
		}
		*ptr = hexutil.Uint64(c.senderNonce)
	case "eth_maxPriorityFeePerGas":
		ptr, ok := result.(*hexutil.Big)
		if !ok {
			return errors.New("result is not a *hexutil.Big")
		}
		*ptr = hexutil.Big(*c.suggestedGasTipCap)
	}
	return nil
}

func (c *stubL1ClientInner) EthSubscribe(ctx context.Context, channel interface{}, args ...interface{}) (*rpc.ClientSubscription, error) {
	return nil, nil
}
func (c *stubL1ClientInner) BatchCallContext(ctx context.Context, b []rpc.BatchElem) error {
	return nil
}
func (c *stubL1ClientInner) Close() {}

func TestFeeAndTipCaps_EnoughBalance_NoBacklog_NoUnconfirmed_BlobTx(t *testing.T) {
	conf := func() *config.DataPosterConfig {
		// Set only the fields that are used by feeAndTipCaps
		// Start with defaults, maybe change for test.
		return &config.DataPosterConfig{
			MaxMempoolTransactions: 18,
			MaxMempoolWeight:       18,
			MinTipCapGwei:          0.05,
			MinBlobTxTipCapGwei:    1,
			MaxTipCapGwei:          5,
			MaxBlobTxTipCapGwei:    10,
			MaxFeeBidMultipleBips:  arbmath.OneInUBips * 10,
			AllocateMempoolBalance: true,

			UrgencyGwei:           2.,
			ElapsedTimeBase:       10 * time.Minute,
			ElapsedTimeImportance: 10,
			TargetPriceGwei:       60.,
		}
	}
	expression, err := govaluate.NewEvaluableExpression(config.DefaultDataPosterConfig.MaxFeeCapFormula)
	if err != nil {
		t.Fatalf("error creating govaluate evaluable expression: %v", err)
	}

	ctx := context.Background()

	p := DataPoster{
		config:           conf,
		extraBacklog:     func() uint64 { return 0 },
		internalState:    state.NewInternalState(nil),
		usingNoOpStorage: false,
		client: ethclient.NewClient(&stubL1ClientInner{
			senderNonce:        1,
			suggestedGasTipCap: big.NewInt(2 * params.GWei),
		}),
		auth: &bind.TransactOpts{
			From: common.Address{},
		},
		maxFeeCapExpression: expression,
		parentChain:         parent.NewParentChain(ctx, big.NewInt(1337), nil),
	}

	var nonce uint64 = 1
	var gasLimit uint64 = 300_000 // reasonable upper bound for mainnet blob batches
	var numBlobs uint64 = 6
	var lastTx *types.Transaction // PostTransaction leaves this nil, used when replacing
	dataCreatedAt := time.Now()
	var dataPosterBacklog uint64 = 0 // Zero backlog for PostTransaction
	var blobGasUsed uint64 = 0xc0000 // 6 blobs of gas
	var excessBlobGas uint64 = 0     // typical current mainnet conditions
	latestHeader := types.Header{
		Number:        big.NewInt(1),
		BaseFee:       big.NewInt(1_000_000_000),
		BlobGasUsed:   &blobGasUsed,
		ExcessBlobGas: &excessBlobGas,
	}

	lockedState := p.internalState.Lock()
	defer p.internalState.Unlock()

	lockedState.Balance = big.NewInt(0).Mul(big.NewInt(params.Ether), big.NewInt(10))

	caps, err := fees.FeeAndTipCaps(ctx, &p, lockedState, nonce, gasLimit, numBlobs, lastTx, dataCreatedAt, dataPosterBacklog, &latestHeader)
	newGasFeeCap, newTipCap, newBlobFeeCap := caps.Fee.NonBlob, caps.Tip, caps.Fee.Blob
	if err != nil {
		t.Fatalf("%s", err)
	}

	// There is no backlog and almost no time elapses since the batch data was
	// created to when it was posted so the maxNormalizedFeeCap is ~60.01 gwei.
	// This is multiplied with the normalizedGas to get targetMaxCost.
	// This is greatly in excess of currentTotalCost * MaxFeeBidMultipleBips,
	// so targetMaxCost is reduced to the current base fee + suggested tip cap +
	// current blob fee multiplied by MaxFeeBidMultipleBips (factor of 10).
	// The blob and non blob factors are then proportionally split out and so
	// the newGasFeeCap is set to (current base fee + suggested tip cap) * 10
	// and newBlobFeeCap is set to current blob gas base fee (1 wei
	// since there is no excess blob gas) * 10.
	expectedGasFeeCap := big.NewInt(30 * params.GWei)
	expectedBlobFeeCap := big.NewInt(10)
	if !arbmath.BigEquals(expectedGasFeeCap, newGasFeeCap) {
		t.Fatalf("feeAndTipCaps didn't return expected gas fee cap. Was: %d, expected: %d", expectedGasFeeCap, newGasFeeCap)
	}
	if !arbmath.BigEquals(expectedBlobFeeCap, newBlobFeeCap) {
		t.Fatalf("feeAndTipCaps didn't return expected blob gas fee cap. Was: %d, expected: %d", expectedBlobFeeCap, newBlobFeeCap)
	}

	// 2 gwei is the amount suggested by the L1 client, so that is the value
	// returned because it doesn't exceed the configured bounds, there is no
	// lastTx to scale against with rbf, and it is not bigger than the computed
	// gasFeeCap.
	expectedTipCap := big.NewInt(2 * params.GWei)
	if !arbmath.BigEquals(expectedTipCap, newTipCap) {
		t.Fatalf("feeAndTipCaps didn't return expected tip cap. Was: %d, expected: %d", expectedTipCap, newTipCap)
	}

	lastBlobTx := &types.BlobTx{}
	err = fees.UpdateTxDataGasCaps(lastBlobTx, newGasFeeCap, newTipCap, newBlobFeeCap)
	if err != nil {
		t.Fatal(err)
	}
	lastTx = types.NewTx(lastBlobTx)
	// Make creation time go backwards so elapsed time increases
	retconnedCreationTime := dataCreatedAt.Add(-time.Minute)
	// Base fee needs to have increased to simulate conditions to not include prev tx
	latestHeader = types.Header{
		Number:        big.NewInt(2),
		BaseFee:       big.NewInt(32_000_000_000),
		BlobGasUsed:   &blobGasUsed,
		ExcessBlobGas: &excessBlobGas,
	}

	caps, err = fees.FeeAndTipCaps(ctx, &p, lockedState, nonce, gasLimit, numBlobs, lastTx, retconnedCreationTime, dataPosterBacklog, &latestHeader)
	newGasFeeCap, newTipCap, newBlobFeeCap = caps.Fee.NonBlob, caps.Tip, caps.Fee.Blob
	_, _, _, _ = newGasFeeCap, newTipCap, newBlobFeeCap, err
	/*
		// I think we expect an increase by *2 due to rbf rules for blob txs,
		// currently appears to be broken since the increase exceeds the
		// current cost (based on current basefees and tip) * config.MaxFeeBidMultipleBips
		// since the previous attempt to send the tx was already using the current cost scaled by
		// the multiple (* 10 bips).
		expectedGasFeeCap = expectedGasFeeCap.Mul(expectedGasFeeCap, big.NewInt(2))
		expectedBlobFeeCap = expectedBlobFeeCap.Mul(expectedBlobFeeCap, big.NewInt(2))
		expectedTipCap = expectedTipCap.Mul(expectedTipCap, big.NewInt(2))

		t.Log("newGasFeeCap", newGasFeeCap, "newTipCap", newTipCap, "newBlobFeeCap", newBlobFeeCap, "err", err)
		if !arbmath.BigEquals(expectedGasFeeCap, newGasFeeCap) {
			t.Fatalf("feeAndTipCaps didn't return expected gas fee cap. Was: %d, expected: %d", expectedGasFeeCap, newGasFeeCap)
		}
		if !arbmath.BigEquals(expectedBlobFeeCap, newBlobFeeCap) {
			t.Fatalf("feeAndTipCaps didn't return expected blob gas fee cap. Was: %d, expected: %d", expectedBlobFeeCap, newBlobFeeCap)
		}
		if !arbmath.BigEquals(expectedTipCap, newTipCap) {
			t.Fatalf("feeAndTipCaps didn't return expected tip cap. Was: %d, expected: %d", expectedTipCap, newTipCap)
		}
	*/

}

func TestFeeAndTipCaps_RBF_RisingBlobFee_FallingBaseFee(t *testing.T) {
	conf := func() *config.DataPosterConfig {
		// Set only the fields that are used by feeAndTipCaps
		// Start with defaults, maybe change for test.
		return &config.DataPosterConfig{
			MaxMempoolTransactions: 18,
			MaxMempoolWeight:       18,
			MinTipCapGwei:          0.05,
			MinBlobTxTipCapGwei:    1,
			MaxTipCapGwei:          5,
			MaxBlobTxTipCapGwei:    10,
			MaxFeeBidMultipleBips:  arbmath.OneInUBips * 10,
			AllocateMempoolBalance: true,

			UrgencyGwei:           2.,
			ElapsedTimeBase:       10 * time.Minute,
			ElapsedTimeImportance: 10,
			TargetPriceGwei:       60.,
		}
	}
	expression, err := govaluate.NewEvaluableExpression(config.DefaultDataPosterConfig.MaxFeeCapFormula)
	if err != nil {
		t.Fatalf("error creating govaluate evaluable expression: %v", err)
	}

	ctx := context.Background()

	p := DataPoster{
		config:           conf,
		extraBacklog:     func() uint64 { return 0 },
		internalState:    state.NewInternalState(nil),
		usingNoOpStorage: false,
		client: ethclient.NewClient(&stubL1ClientInner{
			senderNonce:        1,
			suggestedGasTipCap: big.NewInt(2 * params.GWei),
		}),
		auth: &bind.TransactOpts{
			From: common.Address{},
		},
		maxFeeCapExpression: expression,
		parentChain:         parent.NewParentChain(ctx, big.NewInt(1337), nil),
	}

	var nonce uint64 = 1
	var gasLimit uint64 = 300_000 // reasonable upper bound for mainnet blob batches
	var numBlobs uint64 = 6
	var lastTx *types.Transaction // PostTransaction leaves this nil, used when replacing
	dataCreatedAt := time.Now()
	var dataPosterBacklog uint64 = 0 // Zero backlog for PostTransaction
	var blobGasUsed uint64 = 0xc0000 // 6 blobs of gas
	var excessBlobGas uint64 = 0     // typical current mainnet conditions
	latestHeader := types.Header{
		Number:        big.NewInt(1),
		BaseFee:       big.NewInt(1_000_000_000),
		BlobGasUsed:   &blobGasUsed,
		ExcessBlobGas: &excessBlobGas,
	}

	lockedState := p.internalState.Lock()
	defer p.internalState.Unlock()

	lockedState.Balance = big.NewInt(0).Mul(big.NewInt(params.Ether), big.NewInt(10))

	caps, err := fees.FeeAndTipCaps(ctx, &p, lockedState, nonce, gasLimit, numBlobs, lastTx, dataCreatedAt, dataPosterBacklog, &latestHeader)
	newGasFeeCap, newTipCap, newBlobFeeCap := caps.Fee.NonBlob, caps.Tip, caps.Fee.Blob
	if err != nil {
		t.Fatalf("%s", err)
	}

	// There is no backlog and almost no time elapses since the batch data was
	// created to when it was posted so the maxNormalizedFeeCap is ~60.01 gwei.
	// This is multiplied with the normalizedGas to get targetMaxCost.
	// This is greatly in excess of currentTotalCost * MaxFeeBidMultipleBips,
	// so targetMaxCost is reduced to the current base fee + suggested tip cap +
	// current blob fee multiplied by MaxFeeBidMultipleBips (factor of 10).
	// The blob and non blob factors are then proportionally split out and so
	// the newGasFeeCap is set to (current base fee + suggested tip cap) * 10
	// and newBlobFeeCap is set to current blob gas base fee (1 wei
	// since there is no excess blob gas) * 10.
	expectedGasFeeCap := big.NewInt(30 * params.GWei)
	expectedBlobFeeCap := big.NewInt(10)
	if !arbmath.BigEquals(expectedGasFeeCap, newGasFeeCap) {
		t.Fatalf("feeAndTipCaps didn't return expected gas fee cap. Was: %d, expected: %d", expectedGasFeeCap, newGasFeeCap)
	}
	if !arbmath.BigEquals(expectedBlobFeeCap, newBlobFeeCap) {
		t.Fatalf("feeAndTipCaps didn't return expected blob gas fee cap. Was: %d, expected: %d", expectedBlobFeeCap, newBlobFeeCap)
	}

	// 2 gwei is the amount suggested by the L1 client, so that is the value
	// returned because it doesn't exceed the configured bounds, there is no
	// lastTx to scale against with rbf, and it is not bigger than the computed
	// gasFeeCap.
	expectedTipCap := big.NewInt(2 * params.GWei)
	if !arbmath.BigEquals(expectedTipCap, newTipCap) {
		t.Fatalf("feeAndTipCaps didn't return expected tip cap. Was: %d, expected: %d", expectedTipCap, newTipCap)
	}

	lastBlobTx := &types.BlobTx{}
	err = fees.UpdateTxDataGasCaps(lastBlobTx, newGasFeeCap, newTipCap, newBlobFeeCap)
	if err != nil {
		t.Fatal(err)
	}
	lastTx = types.NewTx(lastBlobTx)
	// Make creation time go backwards so elapsed time increases
	retconnedCreationTime := dataCreatedAt.Add(-time.Minute)
	// Base fee has decreased but blob fee has increased
	blobGasUsed = 0xc0000   // 6 blobs of gas
	excessBlobGas = 8295804 // this should set blob fee to 12 wei
	latestHeader = types.Header{
		Number:        big.NewInt(2),
		BaseFee:       big.NewInt(100_000_000),
		BlobGasUsed:   &blobGasUsed,
		ExcessBlobGas: &excessBlobGas,
	}

	caps, err = fees.FeeAndTipCaps(ctx, &p, lockedState, nonce, gasLimit, numBlobs, lastTx, retconnedCreationTime, dataPosterBacklog, &latestHeader)
	newGasFeeCap, newTipCap, newBlobFeeCap = caps.Fee.NonBlob, caps.Tip, caps.Fee.Blob

	t.Log("newGasFeeCap", newGasFeeCap, "newTipCap", newTipCap, "newBlobFeeCap", newBlobFeeCap, "err", err)
	if arbmath.BigEquals(expectedGasFeeCap, newGasFeeCap) {
		t.Fatalf("feeAndTipCaps didn't return expected gas fee cap. Was: %d, expected NOT: %d", expectedGasFeeCap, newGasFeeCap)
	}
	if arbmath.BigEquals(expectedBlobFeeCap, newBlobFeeCap) {
		t.Fatalf("feeAndTipCaps didn't return expected blob gas fee cap. Was: %d, expected NOT: %d", expectedBlobFeeCap, newBlobFeeCap)
	}
	if arbmath.BigEquals(expectedTipCap, newTipCap) {
		t.Fatalf("feeAndTipCaps didn't return expected tip cap. Was: %d, expected NOT: %d", expectedTipCap, newTipCap)
	}

}

// TestMaybeLogError verifies error classification and ErrorCount management.
// We can verify ErrorCount state changes but NOT which log level was used
// (log functions are package-level, not interceptable).
func TestMaybeLogError(t *testing.T) {
	tests := []struct {
		name          string
		err           error
		initialCount  int // ErrorCount[nonce] before call, -1 means no entry
		expectedCount int // ErrorCount[nonce] after call, -1 means entry deleted
	}{
		{
			name:          "nil error clears error count",
			err:           nil,
			initialCount:  5,
			expectedCount: -1, // entry deleted
		},
		{
			name:          "nil error with no prior entry is no-op",
			err:           nil,
			initialCount:  -1,
			expectedCount: -1,
		},
		{
			name:          "ErrStorageRace increments count from zero",
			err:           storage.ErrStorageRace,
			initialCount:  -1,
			expectedCount: 1,
		},
		{
			name:          "ErrStorageRace increments existing count",
			err:           storage.ErrStorageRace,
			initialCount:  5,
			expectedCount: 6,
		},
		{
			name:          "wrapped ErrStorageRace detected via errors.Is",
			err:           fmt.Errorf("context: %w", storage.ErrStorageRace),
			initialCount:  0,
			expectedCount: 1,
		},
		{
			name:          "ErrStorageRace at threshold (count=20)",
			err:           storage.ErrStorageRace,
			initialCount:  19,
			expectedCount: 20,
		},
		{
			name:          "ErrStorageRace over threshold (count=21)",
			err:           storage.ErrStorageRace,
			initialCount:  20,
			expectedCount: 21,
		},
		{
			name:          "ErrFutureReplacePending increments count",
			err:           legacypool.ErrFutureReplacePending,
			initialCount:  -1,
			expectedCount: 1,
		},
		{
			name:          "ErrFutureReplacePending as substring in wrapped error",
			err:           fmt.Errorf("tx pool: %s", legacypool.ErrFutureReplacePending.Error()),
			initialCount:  3,
			expectedCount: 4,
		},
		{
			name:          "ErrNonceTooHigh increments count",
			err:           core.ErrNonceTooHigh,
			initialCount:  -1,
			expectedCount: 1,
		},
		{
			name:          "ErrNonceTooHigh as substring in wrapped error",
			err:           fmt.Errorf("send failed: %s", core.ErrNonceTooHigh.Error()),
			initialCount:  0,
			expectedCount: 1,
		},
		{
			name:          "non-intermittent error clears existing count",
			err:           errors.New("connection refused"),
			initialCount:  10,
			expectedCount: -1, // entry deleted
		},
		{
			name:          "non-intermittent error with no prior entry",
			err:           errors.New("connection refused"),
			initialCount:  -1,
			expectedCount: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := defaultTestStub()
			dp, is := newTestDataPoster(t, stub, nil)

			s := is.Lock()
			defer is.Unlock()

			nonce := uint64(5)
			tx := makeTestQueuedTx(nonce, big.NewInt(10*params.GWei), big.NewInt(1*params.GWei), true)

			if tt.initialCount >= 0 {
				s.ErrorCount[nonce] = tt.initialCount
			}

			dp.maybeLogError(tt.err, s, tx, "test message")

			gotCount, exists := s.ErrorCount[nonce]
			if tt.expectedCount == -1 {
				if exists {
					t.Errorf("expected ErrorCount[%d] to be deleted, got %d", nonce, gotCount)
				}
			} else {
				if !exists {
					t.Errorf("expected ErrorCount[%d] = %d, but entry was deleted", nonce, tt.expectedCount)
				} else if gotCount != tt.expectedCount {
					t.Errorf("ErrorCount[%d] = %d, want %d", nonce, gotCount, tt.expectedCount)
				}
			}
		})
	}
}

// TestUpdateNonce verifies nonce syncing with L1: header queries, queue
// pruning, ErrorCount cleanup, and finalized vs latest block tag usage.
func TestUpdateNonce(t *testing.T) {
	ctx := context.Background()

	t.Run("same block number is no-op", func(t *testing.T) {
		stub := defaultTestStub()
		stub.latestHeader.Number = big.NewInt(50)
		stub.senderNonce = 10
		dp, is := newTestDataPoster(t, stub, nil)

		s := is.Lock()
		defer is.Unlock()
		s.LastBlock = big.NewInt(50) // same as header
		s.Nonce = 5

		err := dp.updateNonce(ctx, s)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.Nonce != 5 {
			t.Errorf("Nonce = %d, want 5 (unchanged)", s.Nonce)
		}
		if s.LastBlock.Int64() != 50 {
			t.Errorf("LastBlock = %d, want 50 (unchanged)", s.LastBlock.Int64())
		}
	})

	t.Run("nonce increased prunes queue and updates state", func(t *testing.T) {
		stub := defaultTestStub()
		stub.latestHeader.Number = big.NewInt(200)
		stub.senderNonce = 7
		dp, is := newTestDataPoster(t, stub, nil)

		s := is.Lock()
		defer is.Unlock()
		s.LastBlock = big.NewInt(100)
		s.Nonce = 3

		// Put some txs in queue at nonces 3,4,5,6,7
		for i := uint64(3); i <= 7; i++ {
			tx := makeTestQueuedTx(i, big.NewInt(10*params.GWei), big.NewInt(1*params.GWei), true)
			putTxInQueue(t, ctx, s, i, nil, tx)
		}
		// Set error counts for nonces 3,4,5
		s.ErrorCount[3] = 2
		s.ErrorCount[4] = 5
		s.ErrorCount[5] = 1

		err := dp.updateNonce(ctx, s)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.Nonce != 7 {
			t.Errorf("Nonce = %d, want 7", s.Nonce)
		}
		if s.LastBlock.Int64() != 200 {
			t.Errorf("LastBlock = %d, want 200", s.LastBlock.Int64())
		}
		// Error counts for nonces 3,4,5,6 should be deleted (range [s.Nonce, nonce))
		for _, n := range []uint64{3, 4, 5, 6} {
			if _, exists := s.ErrorCount[n]; exists {
				t.Errorf("ErrorCount[%d] should have been deleted", n)
			}
		}
		// Queue should be pruned to nonce-1=6. Item at nonce 6 should still exist
		// (prune is exclusive), and item at nonce 7 should still exist.
		item6, err := s.Queue.Get(ctx, 6)
		if err != nil {
			t.Fatalf("Queue.Get(6): %v", err)
		}
		if item6 == nil {
			t.Error("expected item at nonce 6 to survive pruning (prune to nonce-1)")
		}
		item7, err := s.Queue.Get(ctx, 7)
		if err != nil {
			t.Fatalf("Queue.Get(7): %v", err)
		}
		if item7 == nil {
			t.Error("expected item at nonce 7 to survive pruning")
		}
		// Items at nonces 3,4,5 should be pruned
		for _, n := range []uint64{3, 4, 5} {
			item, err := s.Queue.Get(ctx, n)
			if err != nil {
				t.Fatalf("Queue.Get(%d): %v", n, err)
			}
			if item != nil {
				t.Errorf("expected item at nonce %d to be pruned", n)
			}
		}
	})

	t.Run("nonce equal new block updates LastBlock only", func(t *testing.T) {
		stub := defaultTestStub()
		stub.latestHeader.Number = big.NewInt(200)
		stub.senderNonce = 5
		dp, is := newTestDataPoster(t, stub, nil)

		s := is.Lock()
		defer is.Unlock()
		s.LastBlock = big.NewInt(100) // different from header
		s.Nonce = 5                   // same as on-chain nonce

		err := dp.updateNonce(ctx, s)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.Nonce != 5 {
			t.Errorf("Nonce = %d, want 5 (unchanged)", s.Nonce)
		}
		if s.LastBlock.Int64() != 200 {
			t.Errorf("LastBlock = %d, want 200 (updated to header number)", s.LastBlock.Int64())
		}
	})

	t.Run("nonce decreased is no-op", func(t *testing.T) {
		stub := defaultTestStub()
		stub.latestHeader.Number = big.NewInt(200)
		stub.senderNonce = 3 // less than current
		dp, is := newTestDataPoster(t, stub, nil)

		s := is.Lock()
		defer is.Unlock()
		s.LastBlock = big.NewInt(100)
		s.Nonce = 5

		err := dp.updateNonce(ctx, s)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.Nonce != 5 {
			t.Errorf("Nonce = %d, want 5 (unchanged)", s.Nonce)
		}
		// LastBlock should NOT be updated when nonce < s.Nonce
		if s.LastBlock.Int64() != 100 {
			t.Errorf("LastBlock = %d, want 100 (unchanged)", s.LastBlock.Int64())
		}
	})

	t.Run("NonceAt fails with LastBlock set returns nil", func(t *testing.T) {
		stub := defaultTestStub()
		stub.latestHeader.Number = big.NewInt(200)
		stub.nonceAtErr = errors.New("rpc connection lost")
		dp, is := newTestDataPoster(t, stub, nil)

		s := is.Lock()
		defer is.Unlock()
		s.LastBlock = big.NewInt(100) // non-nil
		s.Nonce = 5

		err := dp.updateNonce(ctx, s)
		if err != nil {
			t.Errorf("expected nil error when NonceAt fails with LastBlock set, got: %v", err)
		}
		// State should be unchanged
		if s.Nonce != 5 {
			t.Errorf("Nonce = %d, want 5", s.Nonce)
		}
	})

	t.Run("NonceAt fails with LastBlock nil returns error", func(t *testing.T) {
		stub := defaultTestStub()
		stub.latestHeader.Number = big.NewInt(200)
		stub.nonceAtErr = errors.New("rpc connection lost")
		dp, is := newTestDataPoster(t, stub, nil)

		s := is.Lock()
		defer is.Unlock()
		s.LastBlock = nil
		s.Nonce = 0

		err := dp.updateNonce(ctx, s)
		if err == nil {
			t.Error("expected error when NonceAt fails with LastBlock nil")
		}
	})

	t.Run("HeaderByNumber fails returns error", func(t *testing.T) {
		stub := defaultTestStub()
		stub.headerByNumberErr = errors.New("header not found")
		dp, is := newTestDataPoster(t, stub, nil)

		s := is.Lock()
		defer is.Unlock()

		err := dp.updateNonce(ctx, s)
		if err == nil {
			t.Error("expected error when HeaderByNumber fails")
		}
	})

	t.Run("waitForL1Finality uses finalized block tag", func(t *testing.T) {
		cfg := config.TestDataPosterConfig
		cfg.WaitForL1Finality = true
		stub := defaultTestStub()
		stub.senderNonce = 0
		dp, is := newTestDataPoster(t, stub, &cfg)

		s := is.Lock()
		defer is.Unlock()

		_ = dp.updateNonce(ctx, s)

		// Verify that eth_getBlockByNumber was called with "finalized" tag
		found := false
		for _, call := range stub.calls {
			if call.Method == "eth_getBlockByNumber" && len(call.Args) >= 1 {
				if tag, ok := call.Args[0].(string); ok && tag == "finalized" {
					found = true
					break
				}
			}
		}
		if !found {
			t.Errorf("expected eth_getBlockByNumber to be called with 'finalized' tag, got calls: %v", stub.calls)
		}
	})

	t.Run("waitForL1Finality false uses latest block tag", func(t *testing.T) {
		cfg := config.TestDataPosterConfig
		cfg.WaitForL1Finality = false
		stub := defaultTestStub()
		stub.senderNonce = 0
		dp, is := newTestDataPoster(t, stub, &cfg)

		s := is.Lock()
		defer is.Unlock()

		_ = dp.updateNonce(ctx, s)

		// Verify that eth_getBlockByNumber was called with "latest" tag
		found := false
		for _, call := range stub.calls {
			if call.Method == "eth_getBlockByNumber" && len(call.Args) >= 1 {
				if tag, ok := call.Args[0].(string); ok && tag == "latest" {
					found = true
					break
				}
			}
		}
		if !found {
			t.Errorf("expected eth_getBlockByNumber to be called with 'latest' tag, got calls: %v", stub.calls)
		}
	})
}

// TestCanPostWithNonce verifies the three posting limits: MaxQueuedTransactions,
// MaxMempoolTransactions, and MaxMempoolWeight.
func TestCanPostWithNonce(t *testing.T) {
	ctx := context.Background()

	t.Run("under all limits returns nil", func(t *testing.T) {
		stub := defaultTestStub()
		stub.senderNonce = 0
		dp, is := newTestDataPoster(t, stub, nil)

		s := is.Lock()
		defer is.Unlock()

		// Empty queue, nonce 0, well under limits
		err := dp.canPostWithNonce(ctx, s, 0, 1)
		if err != nil {
			t.Errorf("expected nil, got: %v", err)
		}
	})

	t.Run("MaxQueuedTransactions exceeded", func(t *testing.T) {
		cfg := config.TestDataPosterConfig
		cfg.MaxQueuedTransactions = 2
		stub := defaultTestStub()
		stub.senderNonce = 0
		dp, is := newTestDataPoster(t, stub, &cfg)

		s := is.Lock()
		defer is.Unlock()

		// Put 2 items in queue (at capacity)
		for i := uint64(0); i < 2; i++ {
			tx := makeTestQueuedTx(i, big.NewInt(10*params.GWei), big.NewInt(1*params.GWei), true)
			putTxInQueue(t, ctx, s, i, nil, tx)
		}

		err := dp.canPostWithNonce(ctx, s, 2, 1)
		if err == nil {
			t.Error("expected error when queue is at MaxQueuedTransactions")
		}
	})

	t.Run("MaxQueuedTransactions zero means unlimited", func(t *testing.T) {
		cfg := config.TestDataPosterConfig
		cfg.MaxQueuedTransactions = 0  // unlimited
		cfg.MaxMempoolTransactions = 0 // don't trigger other checks
		cfg.MaxMempoolWeight = 0
		stub := defaultTestStub()
		stub.senderNonce = 100
		dp, is := newTestDataPoster(t, stub, &cfg)

		s := is.Lock()
		defer is.Unlock()

		// Put many items in queue
		for i := uint64(0); i < 50; i++ {
			tx := makeTestQueuedTx(i, big.NewInt(10*params.GWei), big.NewInt(1*params.GWei), true)
			putTxInQueue(t, ctx, s, i, nil, tx)
		}

		err := dp.canPostWithNonce(ctx, s, 50, 1)
		if err != nil {
			t.Errorf("expected nil with MaxQueuedTransactions=0 (unlimited), got: %v", err)
		}
	})

	t.Run("MaxMempoolTransactions exceeded", func(t *testing.T) {
		cfg := config.TestDataPosterConfig
		cfg.MaxQueuedTransactions = 0 // don't trigger queue limit
		cfg.MaxMempoolTransactions = 5
		stub := defaultTestStub()
		stub.senderNonce = 0 // unconfirmed nonce = 0
		dp, is := newTestDataPoster(t, stub, &cfg)

		s := is.Lock()
		defer is.Unlock()

		// nextNonce=5 >= MaxMempoolTransactions(5) + unconfirmedNonce(0) = 5
		err := dp.canPostWithNonce(ctx, s, 5, 1)
		if err == nil {
			t.Error("expected ErrExceedsMaxMempoolSize")
		} else if !errors.Is(err, ErrExceedsMaxMempoolSize) {
			t.Errorf("expected ErrExceedsMaxMempoolSize, got: %v", err)
		}
	})

	t.Run("MaxMempoolTransactions just under limit", func(t *testing.T) {
		cfg := config.TestDataPosterConfig
		cfg.MaxQueuedTransactions = 0
		cfg.MaxMempoolTransactions = 5
		stub := defaultTestStub()
		stub.senderNonce = 0
		dp, is := newTestDataPoster(t, stub, &cfg)

		s := is.Lock()
		defer is.Unlock()

		// nextNonce=4 < MaxMempoolTransactions(5) + unconfirmedNonce(0) = 5
		err := dp.canPostWithNonce(ctx, s, 4, 1)
		if err != nil {
			t.Errorf("expected nil (under limit), got: %v", err)
		}
	})

	t.Run("MaxMempoolTransactions zero means unlimited", func(t *testing.T) {
		cfg := config.TestDataPosterConfig
		cfg.MaxQueuedTransactions = 0
		cfg.MaxMempoolTransactions = 0 // unlimited
		cfg.MaxMempoolWeight = 0       // also unlimited
		stub := defaultTestStub()
		stub.senderNonce = 0
		dp, is := newTestDataPoster(t, stub, &cfg)

		s := is.Lock()
		defer is.Unlock()

		err := dp.canPostWithNonce(ctx, s, 100, 1)
		if err != nil {
			t.Errorf("expected nil with all limits=0, got: %v", err)
		}
	})

	t.Run("unconfirmedNonce greater than nextNonce", func(t *testing.T) {
		cfg := config.TestDataPosterConfig
		cfg.MaxQueuedTransactions = 0
		cfg.MaxMempoolTransactions = 0 // skip mempool tx check
		cfg.MaxMempoolWeight = 18      // enable weight check
		stub := defaultTestStub()
		stub.senderNonce = 10 // greater than nextNonce
		dp, is := newTestDataPoster(t, stub, &cfg)

		s := is.Lock()
		defer is.Unlock()

		err := dp.canPostWithNonce(ctx, s, 5, 1)
		if err == nil {
			t.Error("expected error when unconfirmedNonce > nextNonce")
		}
	})

	t.Run("MaxMempoolWeight with Post4844Blobs false is effectively no-op", func(t *testing.T) {
		// When Post4844Blobs=false, maxBlobGasPerBlock=0, so
		// weightDiff = min(newWeight-confirmedWeight, 0) = 0, always passes.
		cfg := config.TestDataPosterConfig
		cfg.MaxQueuedTransactions = 0
		cfg.MaxMempoolTransactions = 0
		cfg.MaxMempoolWeight = 1 // very low limit
		cfg.Post4844Blobs = false
		stub := defaultTestStub()
		stub.senderNonce = 0
		dp, is := newTestDataPoster(t, stub, &cfg)

		s := is.Lock()
		defer is.Unlock()

		// Large weight, but Post4844Blobs=false so weight check is effectively disabled
		err := dp.canPostWithNonce(ctx, s, 0, 999999)
		if err != nil {
			t.Errorf("expected nil (weight check is no-op when Post4844Blobs=false), got: %v", err)
		}
	})
}

// TestGetNextNonceAndMaybeMeta verifies nonce determination: queue-based,
// updateNonce-based, and fallback paths.
func TestGetNextNonceAndMaybeMeta(t *testing.T) {
	ctx := context.Background()

	t.Run("queue has items returns next nonce with meta", func(t *testing.T) {
		stub := defaultTestStub()
		stub.senderNonce = 0 // unconfirmed nonce, low enough to pass canPostWithNonce
		dp, is := newTestDataPoster(t, stub, nil)

		s := is.Lock()
		defer is.Unlock()

		meta := []byte("test-meta-data")
		tx := makeTestQueuedTx(5, big.NewInt(10*params.GWei), big.NewInt(1*params.GWei), true)
		tx.Meta = meta
		putTxInQueue(t, ctx, s, 5, nil, tx)

		nonce, gotMeta, hasMeta, cumWeight, err := dp.getNextNonceAndMaybeMeta(ctx, s, 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if nonce != 6 {
			t.Errorf("nonce = %d, want 6 (lastQueuedNonce + 1)", nonce)
		}
		if !hasMeta {
			t.Error("hasMeta = false, want true")
		}
		if string(gotMeta) != string(meta) {
			t.Errorf("meta = %q, want %q", gotMeta, meta)
		}
		// CumulativeWeight: StoredCumulativeWeight is nil, so it falls back to tx.FullTx.Nonce() = 5
		if cumWeight != 5 {
			t.Errorf("cumulativeWeight = %d, want 5 (falls back to nonce)", cumWeight)
		}
	})

	t.Run("queue empty updateNonce succeeds", func(t *testing.T) {
		stub := defaultTestStub()
		stub.latestHeader.Number = big.NewInt(200)
		stub.senderNonce = 3
		dp, is := newTestDataPoster(t, stub, nil)

		s := is.Lock()
		defer is.Unlock()
		s.LastBlock = big.NewInt(0) // different from header, so updateNonce proceeds

		nonce, meta, hasMeta, cumWeight, err := dp.getNextNonceAndMaybeMeta(ctx, s, 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if nonce != 3 {
			t.Errorf("nonce = %d, want 3", nonce)
		}
		if hasMeta {
			t.Error("hasMeta = true, want false (queue was empty)")
		}
		if meta != nil {
			t.Errorf("meta = %v, want nil", meta)
		}
		// When queue is empty, cumulativeWeight == s.Nonce
		if cumWeight != 3 {
			t.Errorf("cumulativeWeight = %d, want 3 (equals s.Nonce)", cumWeight)
		}
	})

	t.Run("queue empty updateNonce fails non-persistent with finality returns error", func(t *testing.T) {
		cfg := config.TestDataPosterConfig
		cfg.WaitForL1Finality = true
		stub := defaultTestStub()
		stub.headerByNumberErr = errors.New("header fetch failed")
		dp, is := newTestDataPoster(t, stub, &cfg)

		s := is.Lock()
		defer is.Unlock()

		// slice storage is non-persistent, waitForL1Finality=true → no fallback
		_, _, _, _, err := dp.getNextNonceAndMaybeMeta(ctx, s, 1)
		if err == nil {
			t.Error("expected error when updateNonce fails with non-persistent queue + waitForL1Finality")
		}
	})

	t.Run("queue empty updateNonce fails non-persistent no finality falls back", func(t *testing.T) {
		cfg := config.TestDataPosterConfig
		cfg.WaitForL1Finality = false
		stub := defaultTestStub()
		stub.latestBlockNumber = 100
		// headerByNumberErr causes updateNonce to fail
		stub.headerByNumberErr = errors.New("header fetch failed")
		// But NonceAt should still work for the fallback path
		stub.senderNonce = 7
		dp, is := newTestDataPoster(t, stub, &cfg)

		s := is.Lock()
		defer is.Unlock()

		nonce, meta, hasMeta, cumWeight, err := dp.getNextNonceAndMaybeMeta(ctx, s, 1)
		if err != nil {
			t.Fatalf("expected fallback to succeed, got: %v", err)
		}
		if nonce != 7 {
			t.Errorf("nonce = %d, want 7 (from fallback NonceAt)", nonce)
		}
		if hasMeta {
			t.Error("hasMeta = true, want false")
		}
		if meta != nil {
			t.Errorf("meta = %v, want nil", meta)
		}
		if cumWeight != 7 {
			t.Errorf("cumulativeWeight = %d, want 7", cumWeight)
		}
		// Verify fallback updated state
		if s.Nonce != 7 {
			t.Errorf("s.Nonce = %d, want 7", s.Nonce)
		}
		// Fallback queries blockNum-1 = 99
		if s.LastBlock == nil || s.LastBlock.Int64() != 99 {
			t.Errorf("s.LastBlock = %v, want 99", s.LastBlock)
		}
	})

	t.Run("canPostWithNonce rejects", func(t *testing.T) {
		cfg := config.TestDataPosterConfig
		cfg.MaxMempoolTransactions = 1 // very restrictive
		stub := defaultTestStub()
		stub.senderNonce = 0
		dp, is := newTestDataPoster(t, stub, &cfg)

		s := is.Lock()
		defer is.Unlock()

		// Put a tx in queue so we go through the "queue has items" path
		tx := makeTestQueuedTx(0, big.NewInt(10*params.GWei), big.NewInt(1*params.GWei), true)
		putTxInQueue(t, ctx, s, 0, nil, tx)

		// nextNonce=1 >= MaxMempoolTransactions(1) + unconfirmedNonce(0) = 1
		_, _, _, _, err := dp.getNextNonceAndMaybeMeta(ctx, s, 1)
		if err == nil {
			t.Error("expected error from canPostWithNonce")
		} else if !errors.Is(err, ErrExceedsMaxMempoolSize) {
			t.Errorf("expected ErrExceedsMaxMempoolSize, got: %v", err)
		}
	})
}

// TestSendTx verifies transaction sending: queue persistence, "already known"
// error handling, nonce gap avoidance, and Sent flag management.
func TestSendTx(t *testing.T) {
	ctx := context.Background()

	t.Run("successful first send", func(t *testing.T) {
		stub := defaultTestStub()
		dp, is := newTestDataPoster(t, stub, nil)

		s := is.Lock()
		defer is.Unlock()

		newTx := makeTestQueuedTx(0, big.NewInt(10*params.GWei), big.NewInt(1*params.GWei), false)

		err := dp.sendTx(ctx, s, nil, newTx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stub.sendTxCount != 1 {
			t.Errorf("sendTxCount = %d, want 1", stub.sendTxCount)
		}
		// After send, saveTx is called again with Sent=true
		queuedItem, err := s.Queue.Get(ctx, 0)
		if err != nil {
			t.Fatalf("Queue.Get(0): %v", err)
		}
		if queuedItem == nil {
			t.Fatal("expected item at nonce 0 in queue")
		}
		if !queuedItem.Sent {
			t.Error("expected Sent=true after successful send")
		}
	})

	t.Run("nonce too low is already known", func(t *testing.T) {
		stub := defaultTestStub()
		stub.sendTxErr = errors.New("nonce too low")
		dp, is := newTestDataPoster(t, stub, nil)

		s := is.Lock()
		defer is.Unlock()

		newTx := makeTestQueuedTx(0, big.NewInt(10*params.GWei), big.NewInt(1*params.GWei), false)

		err := dp.sendTx(ctx, s, nil, newTx)
		if err != nil {
			t.Errorf("expected nil (nonce too low treated as already known), got: %v", err)
		}
		// Should still be marked as Sent
		queuedItem, err := s.Queue.Get(ctx, 0)
		if err != nil {
			t.Fatalf("Queue.Get(0): %v", err)
		}
		if queuedItem == nil || !queuedItem.Sent {
			t.Error("expected Sent=true even for 'nonce too low'")
		}
	})

	t.Run("unknown send error propagated", func(t *testing.T) {
		stub := defaultTestStub()
		stub.sendTxErr = errors.New("connection refused")
		dp, is := newTestDataPoster(t, stub, nil)

		s := is.Lock()
		defer is.Unlock()

		newTx := makeTestQueuedTx(0, big.NewInt(10*params.GWei), big.NewInt(1*params.GWei), false)

		err := dp.sendTx(ctx, s, nil, newTx)
		if err == nil {
			t.Error("expected error to be propagated")
		} else if !strings.Contains(err.Error(), "connection refused") {
			t.Errorf("expected 'connection refused' error, got: %v", err)
		}
		// Item should be in queue but Sent should be false (saveTx with Sent=true not reached)
		queuedItem, err := s.Queue.Get(ctx, 0)
		if err != nil {
			t.Fatalf("Queue.Get(0): %v", err)
		}
		if queuedItem != nil && queuedItem.Sent {
			t.Error("expected Sent=false when send fails with unknown error")
		}
	})

	t.Run("nonce gap different type avoids sending", func(t *testing.T) {
		stub := defaultTestStub()
		stub.latestBlockNumber = 100
		// reorgResistantTxCount at block 99: nonce 0 (low, so nonce 1 > 0 triggers gap avoidance)
		stub.nonceAtFunc = func(tag string) uint64 { return 0 }
		dp, is := newTestDataPoster(t, stub, nil)

		s := is.Lock()
		defer is.Unlock()

		// Preceding tx at nonce 0: DynamicFeeTx, Sent=true
		precedingTx := makeTestQueuedTx(0, big.NewInt(10*params.GWei), big.NewInt(1*params.GWei), true)
		putTxInQueue(t, ctx, s, 0, nil, precedingTx)

		// New tx at nonce 1: BlobTx (different type!), not yet sent
		newTx := makeTestBlobQueuedTx(1, big.NewInt(10*params.GWei), big.NewInt(1*params.GWei), big.NewInt(1*params.GWei), 1, false)

		err := dp.sendTx(ctx, s, nil, newTx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Should NOT have sent (nonce gap avoidance)
		if stub.sendTxCount != 0 {
			t.Errorf("sendTxCount = %d, want 0 (tx should not be sent due to nonce gap)", stub.sendTxCount)
		}
		// Item should be in queue but Sent=false
		queuedItem, err := s.Queue.Get(ctx, 1)
		if err != nil {
			t.Fatalf("Queue.Get(1): %v", err)
		}
		if queuedItem == nil {
			t.Fatal("expected item at nonce 1 in queue")
		}
		if queuedItem.Sent {
			t.Error("expected Sent=false (tx was not sent due to nonce gap)")
		}
	})

	t.Run("same type preceding sent allows sending", func(t *testing.T) {
		stub := defaultTestStub()
		dp, is := newTestDataPoster(t, stub, nil)

		s := is.Lock()
		defer is.Unlock()

		// Preceding tx at nonce 0: DynamicFeeTx, Sent=true
		precedingTx := makeTestQueuedTx(0, big.NewInt(10*params.GWei), big.NewInt(1*params.GWei), true)
		putTxInQueue(t, ctx, s, 0, nil, precedingTx)

		// New tx at nonce 1: same type (DynamicFeeTx), not yet sent
		newTx := makeTestQueuedTx(1, big.NewInt(10*params.GWei), big.NewInt(1*params.GWei), false)

		err := dp.sendTx(ctx, s, nil, newTx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stub.sendTxCount != 1 {
			t.Errorf("sendTxCount = %d, want 1", stub.sendTxCount)
		}
	})

	t.Run("preceding tx nil means already confirmed allows sending", func(t *testing.T) {
		stub := defaultTestStub()
		dp, is := newTestDataPoster(t, stub, nil)

		s := is.Lock()
		defer is.Unlock()

		// No preceding tx at nonce 4 (already pruned/confirmed)
		// New tx at nonce 5
		newTx := makeTestQueuedTx(5, big.NewInt(10*params.GWei), big.NewInt(1*params.GWei), false)

		err := dp.sendTx(ctx, s, nil, newTx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stub.sendTxCount != 1 {
			t.Errorf("sendTxCount = %d, want 1", stub.sendTxCount)
		}
	})
}

func TestEvalMaxFeeCapExpr_EdgeCases(t *testing.T) {
	t.Run("result capped at 1e9 gwei", func(t *testing.T) {
		stub := defaultTestStub()
		dp, _ := newTestDataPoster(t, stub, nil)

		// Huge backlog should produce a massive fee cap, but it's capped at 1e9 * GWei
		result, err := fees.EvalMaxFeeCapExpr(dp, 1e18, 10*time.Hour)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		maxAllowed := new(big.Int).Mul(big.NewInt(1e9), big.NewInt(params.GWei))
		if result.Cmp(maxAllowed) > 0 {
			t.Errorf("result %v exceeds cap of 1e9 gwei (%v)", result, maxAllowed)
		}
	})

	t.Run("zero elapsed time and zero backlog", func(t *testing.T) {
		stub := defaultTestStub()
		dp, _ := newTestDataPoster(t, stub, nil)

		result, err := fees.EvalMaxFeeCapExpr(dp, 0, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// With zero backlog and zero elapsed, the formula reduces to TargetPriceGWei
		// = 60 GWei (from TestDataPosterConfig)
		expectedMin := big.NewInt(59 * params.GWei) // allow some float rounding
		expectedMax := big.NewInt(61 * params.GWei)
		if result.Cmp(expectedMin) < 0 || result.Cmp(expectedMax) > 0 {
			t.Errorf("result %v not in expected range [%v, %v]", result, expectedMin, expectedMax)
		}
	})
}

func TestFeeAndTipCaps_ZeroBalance(t *testing.T) {
	ctx := context.Background()

	t.Run("zero balance new tx returns minimal caps", func(t *testing.T) {
		// With zero balance and no lastTx, the function does NOT return an error.
		// Instead, targetMaxCost is clamped to 0, resulting in feeCap=0 which
		// gets floored to 1 wei at the end (line 739-741).
		// Note: s.Balance defaults to 0 from NewInternalState — that's what
		// feeAndTipCaps reads, not the stub's BalanceAt RPC response.
		stub := defaultTestStub()
		stub.senderNonce = 0
		dp, is := newTestDataPoster(t, stub, nil)

		s := is.Lock()
		defer is.Unlock()

		header := defaultTestHeader()

		caps, err := fees.FeeAndTipCaps(
			ctx, dp, s,
			0,       // nonce
			300_000, // gasLimit
			0,       // numBlobs
			nil,     // lastTx (new tx)
			time.Now(),
			0, // backlog
			header,
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// feeCap is floored to 1 wei
		if caps.Fee.NonBlob.Cmp(big.NewInt(1)) != 0 {
			t.Errorf("feeCap = %v, want 1 (minimum floor)", caps.Fee.NonBlob)
		}
		// tipCap is clamped to feeCap (line 689-696)
		if caps.Tip.Cmp(caps.Fee.NonBlob) > 0 {
			t.Errorf("tipCap %v > feeCap %v (should be clamped)", caps.Tip, caps.Fee.NonBlob)
		}
	})

	t.Run("zero balance replacing returns lastTx caps from feeAndTipCaps", func(t *testing.T) {
		// s.Balance defaults to 0 from NewInternalState.
		stub := defaultTestStub()
		stub.senderNonce = 0
		dp, is := newTestDataPoster(t, stub, nil)

		s := is.Lock()
		defer is.Unlock()

		header := defaultTestHeader()

		lastTx := types.NewTx(&types.DynamicFeeTx{
			Nonce:     0,
			GasTipCap: big.NewInt(5 * params.GWei),
			GasFeeCap: big.NewInt(50 * params.GWei),
			Gas:       300_000,
			To:        nil,
			Value:     big.NewInt(0),
		})

		caps, err := fees.FeeAndTipCaps(
			ctx, dp, s,
			0,       // nonce
			300_000, // gasLimit
			0,       // numBlobs
			lastTx,  // replacing existing tx
			time.Now(),
			0, // backlog
			header,
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// When balance is zero and replacing, should return lastTx's caps
		if caps.Fee.NonBlob.Cmp(lastTx.GasFeeCap()) != 0 {
			t.Errorf("feeCap = %v, want %v (lastTx.GasFeeCap)", caps.Fee.NonBlob, lastTx.GasFeeCap())
		}
		if caps.Tip.Cmp(lastTx.GasTipCap()) != 0 {
			t.Errorf("tipCap = %v, want %v (lastTx.GasTipCap)", caps.Tip, lastTx.GasTipCap())
		}
		// BlobGasFeeCap should be nil for non-blob tx
		if caps.Fee.Blob != nil {
			t.Errorf("blobFeeCap = %v, want nil", caps.Fee.Blob)
		}
	})
}

// TestReplaceTx verifies replace-by-fee: fee threshold checks, tx re-signing,
// DeprecatedData updates, and NextReplacement scheduling.
func TestReplaceTx(t *testing.T) {
	ctx := context.Background()

	t.Run("fee increase above threshold replaces tx", func(t *testing.T) {
		// Set up a prevTx with very low fees and a stub with conditions that
		// produce much higher fee caps (high balance, reasonable basefee).
		stub := defaultTestStub()
		stub.senderNonce = 0
		stub.suggestedGasTipCap = big.NewInt(2 * params.GWei)
		dp, is := newTestDataPoster(t, stub, nil)

		s := is.Lock()
		defer is.Unlock()
		// Must set s.Balance — feeAndTipCaps uses internal state balance, not stub's BalanceAt
		s.Balance = new(big.Int).Mul(big.NewInt(100), big.NewInt(params.Ether))

		// prevTx with very low fees — feeAndTipCaps should compute much higher
		prevTx := makeTestQueuedTx(0, big.NewInt(1*params.GWei), big.NewInt(100_000_000), true)
		prevTx.Created = time.Now().Add(-2 * time.Minute) // some elapsed time
		putTxInQueue(t, ctx, s, 0, nil, prevTx)

		err := dp.replaceTx(ctx, s, prevTx, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should have sent (replacement happened)
		if stub.sendTxCount != 1 {
			t.Errorf("sendTxCount = %d, want 1", stub.sendTxCount)
		}

		// Check that DeprecatedData was updated
		queuedItem, err := s.Queue.Get(ctx, 0)
		if err != nil {
			t.Fatalf("Queue.Get(0): %v", err)
		}
		if queuedItem == nil {
			t.Fatal("expected item at nonce 0")
		}
		// New fee cap should be higher than original 1 GWei
		if queuedItem.FullTx.GasFeeCap().Cmp(big.NewInt(1*params.GWei)) <= 0 {
			t.Errorf("expected replacement to have higher fee cap, got %v", queuedItem.FullTx.GasFeeCap())
		}
		// DeprecatedData should reflect new fees
		if queuedItem.DeprecatedData.GasFeeCap.Cmp(big.NewInt(1*params.GWei)) <= 0 {
			t.Errorf("expected DeprecatedData.GasFeeCap to be updated, got %v", queuedItem.DeprecatedData.GasFeeCap)
		}
		// Sent should be true (sendTx sets it)
		if !queuedItem.Sent {
			t.Error("expected Sent=true after replacement")
		}
	})

	t.Run("fee increase below threshold re-sends original", func(t *testing.T) {
		// Set up prevTx with fees already high enough that feeAndTipCaps
		// doesn't produce a >110% increase.
		stub := defaultTestStub()
		stub.senderNonce = 0
		stub.suggestedGasTipCap = big.NewInt(1 * params.GWei)
		// Set basefee low so fee caps won't be much higher than prevTx's
		stub.latestHeader = defaultTestHeader()
		stub.latestHeader.BaseFee = big.NewInt(1 * params.GWei)
		dp, is := newTestDataPoster(t, stub, nil)

		s := is.Lock()
		defer is.Unlock()
		s.Balance = new(big.Int).Mul(big.NewInt(100), big.NewInt(params.Ether))

		// prevTx with already-high fees (close to what feeAndTipCaps will compute)
		prevTx := makeTestQueuedTx(0, big.NewInt(200*params.GWei), big.NewInt(5*params.GWei), true)
		prevTx.Created = time.Now() // just created, so elapsed time is tiny → low fee formula result
		putTxInQueue(t, ctx, s, 0, nil, prevTx)

		err := dp.replaceTx(ctx, s, prevTx, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should still have sent (re-sends original when below threshold)
		if stub.sendTxCount != 1 {
			t.Errorf("sendTxCount = %d, want 1 (re-sends original)", stub.sendTxCount)
		}

		// The queued item should still have the original fee caps
		// (no replacement happened, just re-send)
		queuedItem, err := s.Queue.Get(ctx, 0)
		if err != nil {
			t.Fatalf("Queue.Get(0): %v", err)
		}
		if queuedItem == nil {
			t.Fatal("expected item at nonce 0")
		}
		// Fee cap should be unchanged from prevTx
		if queuedItem.FullTx.GasFeeCap().Cmp(prevTx.FullTx.GasFeeCap()) != 0 {
			t.Errorf("expected fee cap unchanged at %v, got %v", prevTx.FullTx.GasFeeCap(), queuedItem.FullTx.GasFeeCap())
		}
	})

	t.Run("NextReplacement scheduled from ReplacementTimes", func(t *testing.T) {
		stub := defaultTestStub()
		stub.senderNonce = 0
		stub.suggestedGasTipCap = big.NewInt(2 * params.GWei)
		dp, is := newTestDataPoster(t, stub, nil)

		s := is.Lock()
		defer is.Unlock()
		s.Balance = new(big.Int).Mul(big.NewInt(100), big.NewInt(params.Ether))

		// Created 30 seconds ago — should be in the first replacement window
		prevTx := makeTestQueuedTx(0, big.NewInt(1*params.GWei), big.NewInt(100_000_000), true)
		prevTx.Created = time.Now().Add(-30 * time.Second)
		putTxInQueue(t, ctx, s, 0, nil, prevTx)

		err := dp.replaceTx(ctx, s, prevTx, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		queuedItem, err := s.Queue.Get(ctx, 0)
		if err != nil {
			t.Fatalf("Queue.Get(0): %v", err)
		}
		if queuedItem == nil {
			t.Fatal("expected item at nonce 0")
		}
		// NextReplacement should be set to a future time
		if queuedItem.NextReplacement.Before(time.Now()) {
			t.Errorf("NextReplacement %v should be in the future", queuedItem.NextReplacement)
		}
	})

	t.Run("HeaderByNumber fails returns error", func(t *testing.T) {
		stub := defaultTestStub()
		stub.headerByNumberErr = errors.New("header unavailable")
		dp, is := newTestDataPoster(t, stub, nil)

		s := is.Lock()
		defer is.Unlock()

		prevTx := makeTestQueuedTx(0, big.NewInt(10*params.GWei), big.NewInt(1*params.GWei), true)
		putTxInQueue(t, ctx, s, 0, nil, prevTx)

		err := dp.replaceTx(ctx, s, prevTx, 0)
		if err == nil {
			t.Error("expected error when header unavailable")
		}
	})
}

// TestSaveTx verifies queue persistence: nonce mismatch error, RLP dedup
// (identical tx skip), and normal compare-and-swap replacement.
func TestSaveTx(t *testing.T) {
	ctx := context.Background()

	t.Run("nonce mismatch returns error", func(t *testing.T) {
		stub := defaultTestStub()
		dp, is := newTestDataPoster(t, stub, nil)

		s := is.Lock()
		defer is.Unlock()

		prevTx := makeTestQueuedTx(5, big.NewInt(10*params.GWei), big.NewInt(1*params.GWei), true)
		newTx := makeTestQueuedTx(6, big.NewInt(10*params.GWei), big.NewInt(1*params.GWei), false)

		err := dp.saveTx(ctx, s, prevTx, newTx)
		if err == nil {
			t.Error("expected error for nonce mismatch")
		} else if !strings.Contains(err.Error(), "doesn't match") {
			t.Errorf("expected nonce mismatch error, got: %v", err)
		}
	})

	t.Run("identical prevTx and newTx skips save", func(t *testing.T) {
		stub := defaultTestStub()
		dp, is := newTestDataPoster(t, stub, nil)

		s := is.Lock()
		defer is.Unlock()

		tx := makeTestQueuedTx(0, big.NewInt(10*params.GWei), big.NewInt(1*params.GWei), true)
		putTxInQueue(t, ctx, s, 0, nil, tx)

		// Pass the same tx as both prevTx and newTx — RLP should be identical
		identical := *tx
		err := dp.saveTx(ctx, s, tx, &identical)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Queue should still have exactly the original item (no double-write)
		length, err := s.Queue.Length(ctx)
		if err != nil {
			t.Fatalf("Queue.Length: %v", err)
		}
		if length != 1 {
			t.Errorf("queue length = %d, want 1", length)
		}
	})

	t.Run("different newTx replaces in queue", func(t *testing.T) {
		stub := defaultTestStub()
		dp, is := newTestDataPoster(t, stub, nil)

		s := is.Lock()
		defer is.Unlock()

		prevTx := makeTestQueuedTx(0, big.NewInt(10*params.GWei), big.NewInt(1*params.GWei), false)
		putTxInQueue(t, ctx, s, 0, nil, prevTx)

		newTx := makeTestQueuedTx(0, big.NewInt(20*params.GWei), big.NewInt(2*params.GWei), true)

		err := dp.saveTx(ctx, s, prevTx, newTx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Queue should have the new tx
		item, err := s.Queue.Get(ctx, 0)
		if err != nil {
			t.Fatalf("Queue.Get(0): %v", err)
		}
		if item == nil {
			t.Fatal("expected item at nonce 0")
		}
		if item.FullTx.GasFeeCap().Cmp(big.NewInt(20*params.GWei)) != 0 {
			t.Errorf("expected updated fee cap 20 GWei, got %v", item.FullTx.GasFeeCap())
		}
	})
}

// TestCanPostWithNonce_WeightPath verifies the MaxMempoolWeight path with
// Post4844Blobs=true, exercising weight calculation and MaxBlobGasPerBlock.
func TestCanPostWithNonce_WeightPath(t *testing.T) {
	ctx := context.Background()

	t.Run("weight under limit with blobs", func(t *testing.T) {
		cfg := config.TestDataPosterConfig
		cfg.MaxQueuedTransactions = 0
		cfg.MaxMempoolTransactions = 0
		cfg.MaxMempoolWeight = 18
		cfg.Post4844Blobs = true
		stub := defaultTestStub()
		stub.senderNonce = 0
		dp, is := newTestDataPoster(t, stub, &cfg)

		s := is.Lock()
		defer is.Unlock()

		// Weight of 1 (single blob), well under limit of 18
		err := dp.canPostWithNonce(ctx, s, 0, 1)
		if err != nil {
			t.Errorf("expected nil, got: %v", err)
		}
	})

	t.Run("weight exceeds limit with blobs", func(t *testing.T) {
		cfg := config.TestDataPosterConfig
		cfg.MaxQueuedTransactions = 0
		cfg.MaxMempoolTransactions = 0
		cfg.MaxMempoolWeight = 2
		cfg.Post4844Blobs = true
		stub := defaultTestStub()
		stub.senderNonce = 0
		dp, is := newTestDataPoster(t, stub, &cfg)

		s := is.Lock()
		defer is.Unlock()

		// Put some txs in queue to build up cumulative weight
		for i := uint64(0); i < 3; i++ {
			tx := makeTestBlobQueuedTx(i, big.NewInt(10*params.GWei), big.NewInt(1*params.GWei), big.NewInt(1*params.GWei), 6, true)
			w := uint64(6)
			tx.StoredCumulativeWeight = &w
			putTxInQueue(t, ctx, s, i, nil, tx)
		}

		// nextNonce=3, weight=6 (6 blobs) — should exceed MaxMempoolWeight=2
		err := dp.canPostWithNonce(ctx, s, 3, 6)
		if err == nil {
			t.Error("expected ErrExceedsMaxMempoolSize")
		} else if !errors.Is(err, ErrExceedsMaxMempoolSize) {
			t.Errorf("expected ErrExceedsMaxMempoolSize, got: %v", err)
		}
	})
}

// TestSendTx_NonceGap_UnsentPreceding verifies nonce gap avoidance when the
// preceding tx exists but was not yet sent.
func TestSendTx_NonceGap_UnsentPreceding(t *testing.T) {
	ctx := context.Background()

	t.Run("unsent preceding tx with nonce above reorg resistant count avoids sending", func(t *testing.T) {
		stub := defaultTestStub()
		stub.latestBlockNumber = 100
		// reorgResistantTxCount at block 99 = 0 (low)
		stub.nonceAtFunc = func(tag string) uint64 { return 0 }
		dp, is := newTestDataPoster(t, stub, nil)

		s := is.Lock()
		defer is.Unlock()

		// Preceding tx at nonce 0: same type (DynamicFeeTx), but Sent=false
		precedingTx := makeTestQueuedTx(0, big.NewInt(10*params.GWei), big.NewInt(1*params.GWei), false)
		putTxInQueue(t, ctx, s, 0, nil, precedingTx)

		// New tx at nonce 1: same type, also not sent
		newTx := makeTestQueuedTx(1, big.NewInt(10*params.GWei), big.NewInt(1*params.GWei), false)

		err := dp.sendTx(ctx, s, nil, newTx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Should NOT have sent (preceding unsent, nonce 1 > reorgResistantCount 0)
		if stub.sendTxCount != 0 {
			t.Errorf("sendTxCount = %d, want 0 (nonce gap avoidance)", stub.sendTxCount)
		}
	})

	t.Run("unsent preceding tx with nonce at reorg resistant count allows sending", func(t *testing.T) {
		stub := defaultTestStub()
		stub.latestBlockNumber = 100
		// reorgResistantTxCount at block 99 = 5 (high enough)
		stub.nonceAtFunc = func(tag string) uint64 { return 5 }
		dp, is := newTestDataPoster(t, stub, nil)

		s := is.Lock()
		defer is.Unlock()

		// Preceding tx at nonce 0: Sent=false
		precedingTx := makeTestQueuedTx(0, big.NewInt(10*params.GWei), big.NewInt(1*params.GWei), false)
		putTxInQueue(t, ctx, s, 0, nil, precedingTx)

		// New tx at nonce 1: nonce 1 <= reorgResistantCount 5, so send proceeds
		newTx := makeTestQueuedTx(1, big.NewInt(10*params.GWei), big.NewInt(1*params.GWei), false)

		err := dp.sendTx(ctx, s, nil, newTx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stub.sendTxCount != 1 {
			t.Errorf("sendTxCount = %d, want 1", stub.sendTxCount)
		}
	})
}

// TestSendTx_PrevTxSentBypassesGapCheck verifies that when prevTx.Sent is true,
// the nonce gap check is skipped entirely (the previouslySent flag is set).
func TestSendTx_PrevTxSentBypassesGapCheck(t *testing.T) {
	ctx := context.Background()

	stub := defaultTestStub()
	stub.latestBlockNumber = 100
	// Low reorgResistantTxCount — would trigger gap avoidance if the check ran
	stub.nonceAtFunc = func(tag string) uint64 { return 0 }
	dp, is := newTestDataPoster(t, stub, nil)

	s := is.Lock()
	defer is.Unlock()

	// Preceding tx: different type (DynamicFeeTx), Sent=true
	precedingTx := makeTestQueuedTx(0, big.NewInt(10*params.GWei), big.NewInt(1*params.GWei), true)
	putTxInQueue(t, ctx, s, 0, nil, precedingTx)

	// New tx: BlobTx at nonce 1 (different type from preceding!)
	// Normally this would trigger nonce gap avoidance, but prevTx.Sent=true
	// sets previouslySent=true, which skips the entire gap check.
	newTx := makeTestBlobQueuedTx(1, big.NewInt(10*params.GWei), big.NewInt(1*params.GWei), big.NewInt(1*params.GWei), 1, false)
	prevTxForReplace := makeTestQueuedTx(1, big.NewInt(5*params.GWei), big.NewInt(500_000_000), true) // Sent=true
	putTxInQueue(t, ctx, s, 1, nil, prevTxForReplace)

	err := dp.sendTx(ctx, s, prevTxForReplace, newTx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should have sent — prevTx.Sent=true bypasses the gap check
	if stub.sendTxCount != 1 {
		t.Errorf("sendTxCount = %d, want 1 (prevTx.Sent=true should bypass gap check)", stub.sendTxCount)
	}
}

// TestUpdateNonce_EmptyErrorCount verifies that updateNonce handles an empty
// ErrorCount map without issues (the len(s.ErrorCount) > 0 guard).
func TestUpdateNonce_EmptyErrorCount(t *testing.T) {
	ctx := context.Background()

	stub := defaultTestStub()
	stub.latestHeader.Number = big.NewInt(200)
	stub.senderNonce = 5
	dp, is := newTestDataPoster(t, stub, nil)

	s := is.Lock()
	defer is.Unlock()
	s.LastBlock = big.NewInt(100)
	s.Nonce = 3
	// ErrorCount is empty (default from NewInternalState)

	// Put txs so prune has something to do
	for i := uint64(3); i <= 5; i++ {
		tx := makeTestQueuedTx(i, big.NewInt(10*params.GWei), big.NewInt(1*params.GWei), true)
		putTxInQueue(t, ctx, s, i, nil, tx)
	}

	err := dp.updateNonce(ctx, s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Nonce != 5 {
		t.Errorf("Nonce = %d, want 5", s.Nonce)
	}
	if len(s.ErrorCount) != 0 {
		t.Errorf("ErrorCount should still be empty, got %v", s.ErrorCount)
	}
}

// TestSaveTx_NilPrevTx verifies the first-insert path where prevTx is nil.
func TestSaveTx_NilPrevTx(t *testing.T) {
	ctx := context.Background()

	stub := defaultTestStub()
	dp, is := newTestDataPoster(t, stub, nil)

	s := is.Lock()
	defer is.Unlock()

	newTx := makeTestQueuedTx(0, big.NewInt(10*params.GWei), big.NewInt(1*params.GWei), false)

	err := dp.saveTx(ctx, s, nil, newTx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Item should be in queue
	item, err := s.Queue.Get(ctx, 0)
	if err != nil {
		t.Fatalf("Queue.Get(0): %v", err)
	}
	if item == nil {
		t.Fatal("expected item at nonce 0 after saveTx with nil prevTx")
	}
	if item.FullTx.GasFeeCap().Cmp(big.NewInt(10*params.GWei)) != 0 {
		t.Errorf("fee cap = %v, want 10 GWei", item.FullTx.GasFeeCap())
	}
}
