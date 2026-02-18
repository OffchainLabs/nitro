// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

// Package deploycache provides helpers for caching contract deployments
// on a temporary L1 backend. The resulting genesis alloc can be merged
// into a simulated backend so that expensive creator deployments are
// done once per test binary instead of once per test.
package deploycache

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/catalyst"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/eth/filters"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/node"
	ethparams "github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/util/testhelpers"
)

const deployTimeout = 5 * time.Minute

// StartL1Backend creates a geth node with a simulated beacon from a
// genesis config. The caller must call stack.Start() and eventually
// stack.Close().
func StartL1Backend(stackConfig *node.Config, l1Genesis *core.Genesis) (*node.Node, *eth.Ethereum, *catalyst.SimulatedBeacon, error) {
	stackConfig.DataDir = ""
	stack, err := node.New(stackConfig)
	if err != nil {
		return nil, nil, nil, err
	}

	nodeConf := ethconfig.Defaults
	nodeConf.Preimages = true
	nodeConf.NetworkId = l1Genesis.Config.ChainID.Uint64() //nolint:staticcheck // Config is the correct field for non-Arbitrum L1 chains
	nodeConf.Genesis = l1Genesis
	nodeConf.Miner.Etherbase = l1Genesis.Coinbase
	nodeConf.Miner.PendingFeeRecipient = l1Genesis.Coinbase
	nodeConf.SyncMode = ethconfig.FullSync

	l1backend, err := eth.New(stack, &nodeConf)
	if err != nil {
		stack.Close()
		return nil, nil, nil, err
	}

	simBeacon, err := catalyst.NewSimulatedBeacon(0, common.Address{}, l1backend)
	if err != nil {
		stack.Close()
		return nil, nil, nil, err
	}
	catalyst.RegisterSimulatedBeaconAPIs(stack, simBeacon)
	stack.RegisterLifecycle(simBeacon)

	stack.RegisterAPIs([]rpc.API{{
		Namespace: "eth",
		Service:   filters.NewFilterAPI(filters.NewFilterSystem(l1backend.APIBackend, filters.Config{})),
	}})

	return stack, l1backend, simBeacon, nil
}

// DumpStateToGenesisAlloc dumps the current L1 state into a GenesisAlloc
// that can be used to seed a new backend.
func DumpStateToGenesisAlloc(l1backend *eth.Ethereum) (types.GenesisAlloc, error) {
	statedb, err := l1backend.BlockChain().State()
	if err != nil {
		return nil, fmt.Errorf("getting L1 state: %w", err)
	}
	dump := statedb.RawDump(&state.DumpConfig{})
	alloc := make(types.GenesisAlloc)
	for addrStr, acct := range dump.Accounts {
		addr := common.HexToAddress(addrStr)
		storage := make(map[common.Hash]common.Hash)
		for k, v := range acct.Storage {
			storage[k] = common.HexToHash(v)
		}
		bal, ok := new(big.Int).SetString(acct.Balance, 10)
		if !ok {
			return nil, fmt.Errorf(
				"parsing balance for %s: %q", addrStr, acct.Balance,
			)
		}
		alloc[addr] = types.Account{
			Nonce:   acct.Nonce,
			Balance: bal,
			Code:    acct.Code,
			Storage: storage,
		}
	}
	return alloc, nil
}

// DeployOnTempL1 spins up a temporary L1 chain, runs deployFn to deploy
// contracts, dumps the resulting state, and shuts down.
func DeployOnTempL1[T any](deployFn func(ctx context.Context, client *ethclient.Client, auth *bind.TransactOpts) (T, error)) (types.GenesisAlloc, T, error) {
	ctx, cancel := context.WithTimeout(context.Background(), deployTimeout)
	defer cancel()

	var zero T
	deployerKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, zero, fmt.Errorf("generating deployer key: %w", err)
	}
	deployerAddr := crypto.PubkeyToAddress(deployerKey.PublicKey)

	l1Genesis := core.DeveloperGenesisBlock(15_000_000, &deployerAddr)
	l1Genesis.Coinbase = deployerAddr
	l1Genesis.BaseFee = big.NewInt(50 * ethparams.GWei)

	stackConfig := testhelpers.CreateStackConfigForTest("")
	stack, l1backend, _, err := StartL1Backend(stackConfig, l1Genesis)
	if err != nil {
		return nil, zero, fmt.Errorf("creating L1 backend: %w", err)
	}
	if err := stack.Start(); err != nil {
		stack.Close()
		return nil, zero, fmt.Errorf("starting L1 stack: %w", err)
	}

	client := ethclient.NewClient(stack.Attach())
	deployerAuth, err := bind.NewKeyedTransactorWithChainID(
		deployerKey, l1Genesis.Config.ChainID, //nolint:staticcheck // Config is the correct field for non-Arbitrum L1 chains
	)
	if err != nil {
		stack.Close()
		return nil, zero, fmt.Errorf("creating deployer auth: %w", err)
	}

	result, err := deployFn(ctx, client, deployerAuth)
	if err != nil {
		stack.Close()
		return nil, zero, err
	}

	alloc, err := DumpStateToGenesisAlloc(l1backend)
	if err != nil {
		stack.Close()
		return nil, zero, fmt.Errorf("dumping L1 state: %w", err)
	}

	stack.Close()
	return alloc, result, nil
}
