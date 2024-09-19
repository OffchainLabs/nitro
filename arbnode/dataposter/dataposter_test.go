package dataposter

import (
	"context"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/Knetic/govaluate"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/google/go-cmp/cmp"
	"github.com/holiman/uint256"
	"github.com/offchainlabs/nitro/arbnode/dataposter/externalsignertest"
	"github.com/offchainlabs/nitro/util/arbmath"
)

func signerTestCfg(addr common.Address, url string) (*ExternalSignerCfg, error) {
	cp, err := externalsignertest.CertPaths()
	if err != nil {
		return nil, fmt.Errorf("getting certificates path: %w", err)
	}
	return &ExternalSignerCfg{
		Address:          common.Bytes2Hex(addr.Bytes()),
		URL:              url,
		Method:           externalsignertest.SignerMethod,
		RootCA:           cp.ServerCert,
		ClientCert:       cp.ClientCert,
		ClientPrivateKey: cp.ClientKey,
	}, nil
}

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
	signerCfg, err := signerTestCfg(srv.Address, srv.URL())
	if err != nil {
		t.Fatalf("Error getting signer test config: %v", err)
	}
	ctx := context.Background()
	signer, addr, err := externalSigner(ctx, signerCfg)
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
				got, err := signer(ctx, addr, tc.tx)
				if err != nil {
					t.Fatalf("Error signing transaction with external signer: %v", err)
				}
				want, err := srv.SignerFn(addr, tc.tx)
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
	expression, err := govaluate.NewEvaluableExpression(DefaultDataPosterConfig.MaxFeeCapFormula)
	if err != nil {
		t.Fatalf("Error creating govaluate evaluable expression for calculating default maxFeeCap formula: %v", err)
	}
	config := DefaultDataPosterConfig
	config.TargetPriceGwei = 0
	p := &DataPoster{
		config:              func() *DataPosterConfig { return &config },
		maxFeeCapExpression: expression,
	}
	result, err := p.evalMaxFeeCapExpr(0, 0)
	if err != nil {
		t.Fatalf("Error evaluating MaxFeeCap expression: %v", err)
	}
	if result.Cmp(common.Big0) != 0 {
		t.Fatalf("Unexpected result. Got: %d, want: 0", result)
	}

	result, err = p.evalMaxFeeCapExpr(0, time.Since(time.Time{}))
	if err != nil {
		t.Fatalf("Error evaluating MaxFeeCap expression: %v", err)
	}
	if result.Cmp(big.NewInt(params.GWei)) <= 0 {
		t.Fatalf("Unexpected result. Got: %d, want: >0", result)
	}
}

type stubL1Client struct {
	senderNonce        uint64
	suggestedGasTipCap *big.Int

	// Define most of the required methods that aren't used by feeAndTipCaps
	backends.SimulatedBackend
}

func (c *stubL1Client) NonceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (uint64, error) {
	return c.senderNonce, nil
}

func (c *stubL1Client) SuggestGasTipCap(ctx context.Context) (*big.Int, error) {
	return c.suggestedGasTipCap, nil
}

// Not used but we need to define
func (c *stubL1Client) BlockNumber(ctx context.Context) (uint64, error) {
	return 0, nil
}

func (c *stubL1Client) CallContractAtHash(ctx context.Context, msg ethereum.CallMsg, blockHash common.Hash) ([]byte, error) {
	return []byte{}, nil
}

func (c *stubL1Client) CodeAtHash(ctx context.Context, address common.Address, blockHash common.Hash) ([]byte, error) {
	return []byte{}, nil
}

func (c *stubL1Client) ChainID(ctx context.Context) (*big.Int, error) {
	return nil, nil
}

func (c *stubL1Client) Client() rpc.ClientInterface {
	return nil
}

func (c *stubL1Client) TransactionSender(ctx context.Context, tx *types.Transaction, block common.Hash, index uint) (common.Address, error) {
	return common.Address{}, nil
}

func TestFeeAndTipCaps_EnoughBalance_NoBacklog_NoUnconfirmed_BlobTx(t *testing.T) {
	conf := func() *DataPosterConfig {
		// Set only the fields that are used by feeAndTipCaps
		// Start with defaults, maybe change for test.
		return &DataPosterConfig{
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
	expression, err := govaluate.NewEvaluableExpression(DefaultDataPosterConfig.MaxFeeCapFormula)
	if err != nil {
		t.Fatalf("error creating govaluate evaluable expression: %v", err)
	}

	p := DataPoster{
		config:           conf,
		extraBacklog:     func() uint64 { return 0 },
		balance:          big.NewInt(0).Mul(big.NewInt(params.Ether), big.NewInt(10)),
		usingNoOpStorage: false,
		client: &stubL1Client{
			senderNonce:        1,
			suggestedGasTipCap: big.NewInt(2 * params.GWei),
		},
		auth: &bind.TransactOpts{
			From: common.Address{},
		},
		maxFeeCapExpression: expression,
	}

	ctx := context.Background()
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

	newGasFeeCap, newTipCap, newBlobFeeCap, err := p.feeAndTipCaps(ctx, nonce, gasLimit, numBlobs, lastTx, dataCreatedAt, dataPosterBacklog, &latestHeader)
	if err != nil {
		t.Fatalf("%s", err)
	}

	// There is no backlog and almost no time elapses since the batch data was
	// created to when it was posted so the maxNormalizedFeeCap is ~60.01 gwei.
	// This is multiplied with the normalizedGas to get targetMaxCost.
	// This is greatly in excess of currentTotalCost * MaxFeeBidMultipleBips,
	// so targetMaxCost is reduced to the current base fee + suggested tip cap +
	// current blob fee multipled by MaxFeeBidMultipleBips (factor of 10).
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
	err = updateTxDataGasCaps(lastBlobTx, newGasFeeCap, newTipCap, newBlobFeeCap)
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

	newGasFeeCap, newTipCap, newBlobFeeCap, err = p.feeAndTipCaps(ctx, nonce, gasLimit, numBlobs, lastTx, retconnedCreationTime, dataPosterBacklog, &latestHeader)
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
	conf := func() *DataPosterConfig {
		// Set only the fields that are used by feeAndTipCaps
		// Start with defaults, maybe change for test.
		return &DataPosterConfig{
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
	expression, err := govaluate.NewEvaluableExpression(DefaultDataPosterConfig.MaxFeeCapFormula)
	if err != nil {
		t.Fatalf("error creating govaluate evaluable expression: %v", err)
	}

	p := DataPoster{
		config:           conf,
		extraBacklog:     func() uint64 { return 0 },
		balance:          big.NewInt(0).Mul(big.NewInt(params.Ether), big.NewInt(10)),
		usingNoOpStorage: false,
		client: &stubL1Client{
			senderNonce:        1,
			suggestedGasTipCap: big.NewInt(2 * params.GWei),
		},
		auth: &bind.TransactOpts{
			From: common.Address{},
		},
		maxFeeCapExpression: expression,
	}

	ctx := context.Background()
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

	newGasFeeCap, newTipCap, newBlobFeeCap, err := p.feeAndTipCaps(ctx, nonce, gasLimit, numBlobs, lastTx, dataCreatedAt, dataPosterBacklog, &latestHeader)
	if err != nil {
		t.Fatalf("%s", err)
	}

	// There is no backlog and almost no time elapses since the batch data was
	// created to when it was posted so the maxNormalizedFeeCap is ~60.01 gwei.
	// This is multiplied with the normalizedGas to get targetMaxCost.
	// This is greatly in excess of currentTotalCost * MaxFeeBidMultipleBips,
	// so targetMaxCost is reduced to the current base fee + suggested tip cap +
	// current blob fee multipled by MaxFeeBidMultipleBips (factor of 10).
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
	err = updateTxDataGasCaps(lastBlobTx, newGasFeeCap, newTipCap, newBlobFeeCap)
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

	newGasFeeCap, newTipCap, newBlobFeeCap, err = p.feeAndTipCaps(ctx, nonce, gasLimit, numBlobs, lastTx, retconnedCreationTime, dataPosterBacklog, &latestHeader)

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
