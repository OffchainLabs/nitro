package dataposter

import (
	"context"
	"fmt"
	"math/big"
	"net/http"
	"testing"
	"time"

	"github.com/Knetic/govaluate"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/google/go-cmp/cmp"
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

func TestExternalSigner(t *testing.T) {
	ctx := context.Background()
	httpSrv, srv := externalsignertest.NewServer(ctx, t)
	t.Cleanup(func() {
		if err := httpSrv.Shutdown(ctx); err != nil {
			t.Fatalf("Error shutting down http server: %v", err)
		}
	})
	cert, key := "./testdata/localhost.crt", "./testdata/localhost.key"
	go func() {
		fmt.Println("Server is listening on port 1234...")
		if err := httpSrv.ListenAndServeTLS(cert, key); err != nil && err != http.ErrServerClosed {
			t.Errorf("ListenAndServeTLS() unexpected error:  %v", err)
			return
		}
	}()
	signerCfg, err := signerTestCfg(srv.Address)
	if err != nil {
		t.Fatalf("Error getting signer test config: %v", err)
	}
	signer, addr, err := externalSigner(ctx, signerCfg)
	if err != nil {
		t.Fatalf("Error getting external signer: %v", err)
	}
	tx := types.NewTx(
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
	got, err := signer(ctx, addr, tx)
	if err != nil {
		t.Fatalf("Error signing transaction with external signer: %v", err)
	}
	args, err := txToSendTxArgs(addr, tx)
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
