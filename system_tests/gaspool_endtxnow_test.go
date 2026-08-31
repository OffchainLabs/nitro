// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/triedb"

	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbos/l2pricing"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/statetransfer"
	"github.com/offchainlabs/nitro/util/testhelpers/env"
)

// newRetryableTestState builds a fresh ArbOS state and its genesis header, for
// driving arbos.ProduceBlock directly.
func newRetryableTestState(t *testing.T) (*state.StateDB, *types.Header, core.ChainContext, *params.ChainConfig) {
	t.Helper()
	executionDB := rawdb.NewMemoryDatabase()
	chainConfig := chaininfo.ArbitrumDevTestChainConfig()
	serializedChainConfig, err := json.Marshal(chainConfig)
	if err != nil {
		t.Fatalf("marshalling chain config: %v", err)
	}
	initMessage := &arbostypes.ParsedInitMessage{
		ChainId:               chainConfig.ChainID,
		InitialL1BaseFee:      arbostypes.DefaultInitialL1BaseFee,
		ChainConfig:           chainConfig,
		SerializedChainConfig: serializedChainConfig,
	}
	options := core.DefaultConfig().WithStateScheme(env.GetTestStateScheme())
	stateRoot, err := arbosState.InitializeArbosInDatabase(
		executionDB,
		options,
		statetransfer.NewMemoryInitDataReader(&statetransfer.ArbosInitializationInfo{}),
		chainConfig,
		nil,
		initMessage,
		0,
		0,
	)
	if err != nil {
		t.Fatalf("initializing arbos: %v", err)
	}
	statedb, err := state.New(stateRoot, state.NewDatabase(triedb.NewDatabase(executionDB, options.TriedbConfig()), nil))
	if err != nil {
		t.Fatalf("opening statedb: %v", err)
	}
	genesis := arbosState.MakeGenesisBlock(common.Hash{}, 0, 0, stateRoot, chainConfig)
	return statedb, genesis.Header(), noopChainContext{chainConfig: chainConfig}, chainConfig
}

// submitRetryableMessage builds the delayed-inbox message an L1
// createRetryableTicket produces, with the retry's declared gas limit set to
// gasLimit. The deposit covers gasLimit*maxFeePerGas so the submission hook
// schedules the auto-redeem.
func submitRetryableMessage(sender common.Address, gasLimit uint64, maxFeePerGas *big.Int) *arbostypes.L1IncomingMessage {
	gasCost := new(big.Int).Mul(new(big.Int).SetUint64(gasLimit), maxFeePerGas)
	// A little headroom above the gas cost for the submission fee.
	deposit := new(big.Int).Add(gasCost, big.NewInt(params.Ether))

	word := func(v *big.Int) []byte { return common.BigToHash(v).Bytes() }
	var body []byte
	body = append(body, common.BytesToHash(sender.Bytes()).Bytes()...) // retryTo
	body = append(body, word(common.Big0)...)                          // callvalue
	body = append(body, word(deposit)...)                              // depositValue
	body = append(body, word(big.NewInt(params.Ether))...)             // maxSubmissionFee
	body = append(body, common.BytesToHash(sender.Bytes()).Bytes()...) // feeRefundAddress
	body = append(body, common.BytesToHash(sender.Bytes()).Bytes()...) // callvalueRefundAddress
	body = append(body, word(new(big.Int).SetUint64(gasLimit))...)     // gasLimit
	body = append(body, word(maxFeePerGas)...)                         // maxFeePerGas
	body = append(body, word(common.Big0)...)                          // dataLength

	requestId := common.BigToHash(big.NewInt(1))
	return &arbostypes.L1IncomingMessage{
		Header: &arbostypes.L1IncomingMessageHeader{
			Kind:        arbostypes.L1MessageType_SubmitRetryable,
			Poster:      sender,
			BlockNumber: 1,
			Timestamp:   1,
			RequestId:   &requestId,
			L1BaseFee:   big.NewInt(0),
		},
		L2msg: body,
	}
}

func txTypes(block *types.Block) []uint8 {
	types := make([]uint8, 0, len(block.Transactions()))
	for _, tx := range block.Transactions() {
		types = append(types, tx.Type())
	}
	return types
}

func countTxType(block *types.Block, txType uint8) int {
	n := 0
	for _, tx := range block.Transactions() {
		if tx.Type() == txType {
			n++
		}
	}
	return n
}

// TestRetryableAutoRedeemSurvivesLargeGasLimit is the regression test for
// NM-GASPOOL-ENDTXNOW-DIVERGE-001 (NIT-5464).
//
// A retryable submission is an endTxNow transaction whose reported gas is the
// declared gas limit of the retry it schedules, not gas the submission itself
// consumed. When that reported gas was debited from the shared block gas pool,
// the auto-redeem the block builder runs immediately afterwards reserved the
// same gas a second time, so a declared limit above half the pool dropped the
// redeem, and a limit above the pool dropped the submission too. Both outcomes
// derive a different block than the v3.11.x line does for the same
// force-included message.
func TestRetryableAutoRedeemSurvivesLargeGasLimit(t *testing.T) {
	maxFeePerGas := big.NewInt(l2pricing.InitialMinimumBaseFeeWei)
	sender := common.HexToAddress("0x1111111111111111111111111111111111111111")

	// wantRedeem records the v3.11.x derivation for each declared limit: the
	// auto-redeem is included whenever its gas fits the block pool, which it
	// does right up to the pool size. Only past the pool does the redeem's own
	// reservation legitimately fail, and even then the submission stays in.
	for _, tc := range []struct {
		name       string
		gasLimit   uint64
		wantRedeem bool
	}{
		// Below half the pool: never affected by the double reservation.
		// Guards against the fix breaking the common case.
		{"small", 1 << 20, true},
		// Above half the pool: the redeem's reservation used to fail here,
		// because the submission had already reserved the same gas.
		{"above half the pool", l2pricing.GethBlockGasLimit/2 + 1, true},
		// Exactly the pool: the redeem reserves the whole pool and still fits.
		{"at the pool", l2pricing.GethBlockGasLimit, true},
		// Past the pool: the redeem cannot fit on any build, but the
		// submission's own debit used to fail too, dropping it as well.
		{"above the pool", l2pricing.GethBlockGasLimit + 1, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			statedb, lastHeader, chainContext, _ := newRetryableTestState(t)
			msg := submitRetryableMessage(sender, tc.gasLimit, maxFeePerGas)

			block, _, _, err := arbos.ProduceBlock(
				msg, 1, lastHeader, statedb, chainContext, false,
				core.NewMessageCommitContext(nil), false,
			)
			if err != nil {
				t.Fatalf("ProduceBlock: %v", err)
			}

			// The submission must be included regardless of its declared limit:
			// the gas it reports belongs to the retry, not to itself.
			if n := countTxType(block, types.ArbitrumSubmitRetryableTxType); n != 1 {
				t.Errorf("submission txs = %d, want 1 (block tx types: %v)", n, txTypes(block))
			}
			wantRedeems := 0
			if tc.wantRedeem {
				wantRedeems = 1
			}
			if n := countTxType(block, types.ArbitrumRetryTxType); n != wantRedeems {
				t.Errorf("auto-redeem txs = %d, want %d (block tx types: %v)", n, wantRedeems, txTypes(block))
			}

			// header.GasUsed counts the submission's reported gas, plus the
			// redeem's own consumption when the redeem ran.
			switch got := block.GasUsed(); {
			case tc.wantRedeem && got <= tc.gasLimit:
				t.Errorf("header.GasUsed = %d, want more than the declared limit %d (the redeem's gas is missing)", got, tc.gasLimit)
			case !tc.wantRedeem && got != tc.gasLimit:
				t.Errorf("header.GasUsed = %d, want exactly the submission's reported gas %d", got, tc.gasLimit)
			}
		})
	}
}
