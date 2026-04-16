// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethhook

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/precompiles"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

type TestChainContext struct {
	chainConfig *params.ChainConfig
}

func (r *TestChainContext) CurrentHeader() *types.Header {
	return &types.Header{}
}

func (r *TestChainContext) GetHeaderByNumber(number uint64) *types.Header {
	return &types.Header{}
}

func (r *TestChainContext) GetHeaderByHash(hash common.Hash) *types.Header {
	return &types.Header{}
}

func (r *TestChainContext) Engine() consensus.Engine {
	return arbos.Engine{}
}

func (r *TestChainContext) Config() *params.ChainConfig { return r.chainConfig }

func (r *TestChainContext) GetHeader(hash common.Hash, num uint64) *types.Header {
	return &types.Header{}
}

var testChainConfig = &params.ChainConfig{
	ChainID:             big.NewInt(0),
	HomesteadBlock:      big.NewInt(0),
	DAOForkBlock:        nil,
	DAOForkSupport:      true,
	EIP150Block:         big.NewInt(0),
	EIP155Block:         big.NewInt(0),
	EIP158Block:         big.NewInt(0),
	ByzantiumBlock:      big.NewInt(0),
	ConstantinopleBlock: big.NewInt(0),
	PetersburgBlock:     big.NewInt(0),
	IstanbulBlock:       big.NewInt(0),
	MuirGlacierBlock:    big.NewInt(0),
	BerlinBlock:         big.NewInt(0),
	LondonBlock:         big.NewInt(0),
	ArbitrumChainParams: chaininfo.ArbitrumDevTestParams(),
}

func TestEthDepositMessage(t *testing.T) {

	_, statedb := arbosState.NewArbosMemoryBackedArbOSState()
	addr := common.HexToAddress("0x32abcdeffffff")
	balance := common.BigToHash(big.NewInt(789789897789798))
	balance2 := common.BigToHash(big.NewInt(98))

	if statedb.GetBalance(addr).Sign() != 0 {
		Fail(t)
	}

	firstRequestId := common.BigToHash(big.NewInt(3))
	header := arbostypes.L1IncomingMessageHeader{
		Kind:        arbostypes.L1MessageType_EthDeposit,
		Poster:      addr,
		BlockNumber: 864513,
		Timestamp:   8794561564,
		RequestId:   &firstRequestId,
		L1BaseFee:   big.NewInt(10000000000000),
	}
	msgBuf := bytes.Buffer{}
	if err := util.AddressToWriter(addr, &msgBuf); err != nil {
		t.Error(err)
	}
	if err := util.HashToWriter(balance, &msgBuf); err != nil {
		t.Error(err)
	}
	msg := arbostypes.L1IncomingMessage{
		Header: &header,
		L2msg:  msgBuf.Bytes(),
	}

	serialized, err := msg.Serialize()
	if err != nil {
		t.Error(err)
	}

	secondRequestId := common.BigToHash(big.NewInt(4))
	header.RequestId = &secondRequestId
	header.Poster = util.RemapL1Address(addr)
	msgBuf2 := bytes.Buffer{}
	if err := util.AddressToWriter(addr, &msgBuf2); err != nil {
		t.Error(err)
	}
	if err := util.HashToWriter(balance2, &msgBuf2); err != nil {
		t.Error(err)
	}
	msg2 := arbostypes.L1IncomingMessage{
		Header: &header,
		L2msg:  msgBuf2.Bytes(),
	}
	serialized2, err := msg2.Serialize()
	if err != nil {
		t.Error(err)
	}

	RunMessagesThroughAPI(t, [][]byte{serialized, serialized2}, statedb)

	balanceAfter := statedb.GetBalance(addr).ToBig()
	if balanceAfter.Cmp(new(big.Int).Add(balance.Big(), balance2.Big())) != 0 {
		Fail(t)
	}
}

func RunMessagesThroughAPI(t *testing.T, msgs [][]byte, statedb *state.StateDB) {
	chainId := big.NewInt(6456554)
	for _, data := range msgs {
		msg, err := arbostypes.ParseIncomingL1Message(bytes.NewReader(data), nil)
		if err != nil {
			t.Error(err)
		}
		txes, err := arbos.ParseL2Transactions(msg, chainId, params.MaxDebugArbosVersionSupported)
		if err != nil {
			t.Error(err)
		}
		chainContext := &TestChainContext{chainConfig: testChainConfig}
		header := &types.Header{
			Number:     big.NewInt(1000),
			Difficulty: big.NewInt(1000),
		}
		blockContext := core.NewEVMBlockContext(header, chainContext, nil)
		evm := vm.NewEVM(blockContext, statedb, testChainConfig, vm.Config{})
		gasPool := core.GasPool(100000)
		for _, tx := range txes {
			_, _, err := core.ApplyTransaction(evm, &gasPool, statedb, header, tx, &header.GasUsed)
			if err != nil {
				Fail(t, err)
			}
		}

		arbos.FinalizeBlock(nil, nil, statedb, testChainConfig)
	}
}

func Require(t *testing.T, err error, printables ...interface{}) {
	t.Helper()
	testhelpers.RequireImpl(t, err, printables...)
}

func Fail(t *testing.T, printables ...interface{}) {
	t.Helper()
	testhelpers.FailImpl(t, printables...)
}

func TestPrecompileBucketMembership(t *testing.T) {
	// Each PrecompiledContractsStartingFromArbOS<N> map is the active-precompile
	// set for the ArbOS version range [N, nextN). A precompile with ArbosVersion()
	// V is registered in a bucket iff V < nextN — i.e. the precompile has at
	// least one active method during the range.
	//
	// If a new ArbOS version is introduced, add a new bucket entry below and bump
	// maxKnownArbosVersion.
	const maxKnownArbosVersion = params.ArbosVersion_60

	if params.MaxDebugArbosVersionSupported > maxKnownArbosVersion {
		t.Errorf("MaxDebugArbosVersionSupported (%d) > maxKnownArbosVersion (%d); add a new bucket and bump the constant",
			params.MaxDebugArbosVersionSupported, maxKnownArbosVersion)
	}

	buckets := []struct {
		name       string
		contracts  map[common.Address]vm.PrecompiledContract
		addrs      []common.Address
		upperBound uint64 // exclusive
	}{
		{"BeforeArbOS30", vm.PrecompiledContractsBeforeArbOS30, vm.PrecompiledAddressesBeforeArbOS30, params.ArbosVersion_30},
		{"StartingFromArbOS30", vm.PrecompiledContractsStartingFromArbOS30, vm.PrecompiledAddressesStartingFromArbOS30, params.ArbosVersion_50},
		{"StartingFromArbOS50", vm.PrecompiledContractsStartingFromArbOS50, vm.PrecompiledAddressesStartingFromArbOS50, params.ArbosVersion_60},
		{"StartingFromArbOS60", vm.PrecompiledContractsStartingFromArbOS60, vm.PrecompiledAddressesStartingFromArbOS60, maxKnownArbosVersion + 1},
	}

	for addr, p := range precompiles.Precompiles() {
		name := p.Precompile().Name()
		v := p.Precompile().ArbosVersion()
		if v > maxKnownArbosVersion {
			t.Errorf("precompile %s has ArbosVersion %d > maxKnownArbosVersion %d; add a new bucket and bump the constant",
				name, v, maxKnownArbosVersion)
			continue
		}
		for _, b := range buckets {
			_, present := b.contracts[addr]
			want := v < b.upperBound
			if present != want {
				t.Errorf("precompile %s (v=%d) in bucket %s: got present=%v, want=%v",
					name, v, b.name, present, want)
			}
		}
	}

	// Ethereum precompile subsets init() explicitly merges into each bucket
	// (see geth-hook.go). Keys must match the bucket names declared above.
	ethSubsets := map[string][]map[common.Address]vm.PrecompiledContract{
		"BeforeArbOS30":       {vm.PrecompiledContractsBerlin},
		"StartingFromArbOS30": {vm.PrecompiledContractsCancun, vm.PrecompiledContractsP256Verify},
		"StartingFromArbOS50": {vm.PrecompiledContractsOsaka},
		"StartingFromArbOS60": {vm.PrecompiledContractsOsaka},
	}

	for _, b := range buckets {
		// Every address from each assigned Ethereum subset must be present.
		ethUnion := make(map[common.Address]struct{})
		for _, subset := range ethSubsets[b.name] {
			for addr := range subset {
				ethUnion[addr] = struct{}{}
				if _, ok := b.contracts[addr]; !ok {
					t.Errorf("bucket %s missing Ethereum precompile %s", b.name, addr.Hex())
				}
			}
		}

		// Total-size closure: bucket = arbos-in-bucket + eth-subsets-union.
		// Catches accidental extras (e.g. a stray addPrecompiles line).
		arbosInBucket := 0
		for _, p := range precompiles.Precompiles() {
			if p.Precompile().ArbosVersion() < b.upperBound {
				arbosInBucket++
			}
		}
		if got, want := len(b.contracts), arbosInBucket+len(ethUnion); got != want {
			t.Errorf("bucket %s has %d entries, expected %d (%d arbos + %d ethereum)",
				b.name, got, want, arbosInBucket, len(ethUnion))
		}
	}

	// Address slice must match its contracts map (catches addAddresses drift).
	for _, b := range buckets {
		if len(b.addrs) != len(b.contracts) {
			t.Errorf("bucket %s: addresses slice has %d entries but contracts map has %d",
				b.name, len(b.addrs), len(b.contracts))
		}
		seen := make(map[common.Address]bool, len(b.addrs))
		for _, a := range b.addrs {
			if seen[a] {
				t.Errorf("bucket %s: address %s appears twice in addresses slice", b.name, a.Hex())
			}
			seen[a] = true
			if _, ok := b.contracts[a]; !ok {
				t.Errorf("bucket %s: address %s in slice but not in contracts map", b.name, a.Hex())
			}
		}
	}
}
