package dataposter

import (
	"context"
	"fmt"
	"math/big"
	"net/http"
	"testing"
	"time"

	"github.com/Knetic/govaluate"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/google/go-cmp/cmp"
	"github.com/holiman/uint256"
	"github.com/offchainlabs/nitro/arbnode/dataposter/externalsigner"
	"github.com/offchainlabs/nitro/arbnode/dataposter/externalsignertest"
)

func TestParseReplacementTimes(t *testing.T) {
	for _, tc := range []struct {
		desc, replacementTimes string
		want                   []time.Duration
		wantErr                bool
	}{
		{
			desc:             "valid case",
			replacementTimes: "1s,2s,1m,5m",
			want: []time.Duration{
				time.Duration(time.Second),
				time.Duration(2 * time.Second),
				time.Duration(time.Minute),
				time.Duration(5 * time.Minute),
				time.Duration(time.Hour * 24 * 365 * 10),
			},
		},
		{
			desc:             "non-increasing replacement times",
			replacementTimes: "1s,2s,1m,5m,1s",
			wantErr:          true,
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			got, err := parseReplacementTimes(tc.replacementTimes)
			if gotErr := (err != nil); gotErr != tc.wantErr {
				t.Fatalf("Got error: %t, want: %t", gotErr, tc.wantErr)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("parseReplacementTimes(%s) unexpected diff:\n%s", tc.replacementTimes, diff)
			}
		})
	}
}

func signerTestCfg(addr common.Address) (*ExternalSignerCfg, error) {
	cp, err := externalsignertest.CertPaths()
	if err != nil {
		return nil, fmt.Errorf("getting certificates path: %w", err)
	}
	return &ExternalSignerCfg{
		Address:          common.Bytes2Hex(addr.Bytes()),
		URL:              externalsignertest.SignerURL,
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
	httpSrv, srv := externalsignertest.NewServer(t)
	cert, key := "./testdata/localhost.crt", "./testdata/localhost.key"
	go func() {
		if err := httpSrv.ListenAndServeTLS(cert, key); err != nil && err != http.ErrServerClosed {
			t.Errorf("ListenAndServeTLS() unexpected error:  %v", err)
			return
		}
	}()
	signerCfg, err := signerTestCfg(srv.Address)
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
				args, err := externalsigner.TxToSignTxArgs(addr, tc.tx)
				if err != nil {
					t.Fatalf("Error converting transaction to sendTxArgs: %v", err)
				}
				want, err := srv.SignerFn(addr, args.ToTransaction())
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
	nonce     uint64
	gasTipCap *big.Int

	// Define most of the required methods that aren't used by feeAndTipCaps
	backends.SimulatedBackend
}

func (c *stubL1Client) NonceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (uint64, error) {
	return c.nonce, nil
}

func (c *stubL1Client) SuggestGasTipCap(ctx context.Context) (*big.Int, error) {
	return c.gasTipCap, nil
}

// Not used but we need to define
func (c *stubL1Client) BlockNumber(ctx context.Context) (uint64, error) {
	return 0, nil
}

func (c *stubL1Client) CallContractAtHash(ctx context.Context, msg ethereum.CallMsg, blockHash common.Hash) ([]byte, error) {
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

func TestFeeAndTipCaps(t *testing.T) {
	conf := func() *DataPosterConfig {
		return &DataPosterConfig{
			MaxMempoolTransactions: 0,
			MaxMempoolWeight:       0,
			MinTipCapGwei:          0,
			MinBlobTxTipCapGwei:    0,
			MaxTipCapGwei:          0,
			MaxBlobTxTipCapGwei:    0,
			MaxFeeBidMultipleBips:  0,
			AllocateMempoolBalance: false,
		}
	}
	expression, err := govaluate.NewEvaluableExpression(DefaultDataPosterConfig.MaxFeeCapFormula)
	if err != nil {
		t.Fatalf("error creating govaluate evaluable expression: %v", err)
	}

	p := DataPoster{
		config:           conf,
		extraBacklog:     func() uint64 { return 0 },
		balance:          big.NewInt(1_000_000_000_000_000_000),
		usingNoOpStorage: false,
		client: &stubL1Client{
			nonce:     1,
			gasTipCap: big.NewInt(2_000_000_000),
		},
		auth: &bind.TransactOpts{
			From: common.Address{},
		},
		maxFeeCapExpression: expression,
	}

	ctx := context.Background()
	var nonce uint64 = 1
	var gasLimit uint64 = 30_000_000
	var numBlobs uint64 = 0
	var lastTx *types.Transaction
	dataCreatedAt := time.Now()
	var dataPosterBacklog uint64 = 0
	var blobGasUsed uint64 = 100
	var excessBlobGas uint64 = 100
	latestHeader := types.Header{
		Number:        big.NewInt(1),
		BaseFee:       big.NewInt(1_000_000_000),
		BlobGasUsed:   &blobGasUsed,
		ExcessBlobGas: &excessBlobGas,
	}

	newGasFeeCap, newTipCap, newBlobFeeCap, err := p.feeAndTipCaps(ctx, nonce, gasLimit, numBlobs, lastTx, dataCreatedAt, dataPosterBacklog, &latestHeader)
	_, _, _, _ = newGasFeeCap, newTipCap, newBlobFeeCap, err

}
