package arbtest

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/arbitrum"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/offchainlabs/arbstate/arbos"
	"github.com/offchainlabs/arbstate/arbstate"
)

func CreateTestBackendWithBalance(t *testing.T) (*arbitrum.Backend, *ethclient.Client, *ecdsa.PrivateKey) {
	arbstate.RequireHookedGeth()
	stackConf := node.DefaultConfig
	var err error
	stackConf.DataDir = t.TempDir()
	defer os.RemoveAll(stackConf.DataDir)
	stackConf.HTTPHost = "localhost"
	stackConf.HTTPModules = append(stackConf.HTTPModules, "eth")
	stack, err := node.New(&stackConf)
	if err != nil {
		if err != nil {
			utils.Fatalf("Error creating protocol stack: %v\n", err)
		}
	}
	nodeConf := ethconfig.Defaults
	nodeConf.NetworkId = arbos.ChainConfig.ChainID.Uint64()

	ownerKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	ownerAddress := crypto.PubkeyToAddress(ownerKey.PublicKey)

	genesisAlloc := make(map[common.Address]core.GenesisAccount)
	genesisAlloc[ownerAddress] = core.GenesisAccount{
		Balance:    big.NewInt(params.Ether),
		Nonce:      0,
		PrivateKey: nil,
	}
	nodeConf.Genesis = &core.Genesis{
		Config:     arbos.ChainConfig,
		Nonce:      0,
		Timestamp:  1633932474,
		ExtraData:  []byte("ArbitrumMainnet"),
		GasLimit:   0,
		Difficulty: big.NewInt(1),
		Mixhash:    common.Hash{},
		Coinbase:   common.Address{},
		Alloc:      genesisAlloc,
		Number:     0,
		GasUsed:    0,
		ParentHash: common.Hash{},
		BaseFee:    big.NewInt(0),
	}

	chainDb, err := stack.OpenDatabaseWithFreezer("chaindata", nodeConf.DatabaseCache, nodeConf.DatabaseHandles, nodeConf.DatabaseFreezer, "eth/db/chaindata/", false)
	if err != nil {
		t.Fatal(err)
	}
	engine := arbos.Engine{
		IsSequencer: true,
	}
	chainConfig, _, genesisErr := core.SetupGenesisBlockWithOverride(chainDb, nodeConf.Genesis, nodeConf.OverrideLondon)
	if _, ok := genesisErr.(*params.ConfigCompatError); genesisErr != nil && !ok {
		t.Fatal(genesisErr)
	}

	vmConfig := vm.Config{
		EnablePreimageRecording: nodeConf.EnablePreimageRecording,
	}
	cacheConfig := &core.CacheConfig{
		TrieCleanLimit:      nodeConf.TrieCleanCache,
		TrieCleanJournal:    stack.ResolvePath(nodeConf.TrieCleanCacheJournal),
		TrieCleanRejournal:  nodeConf.TrieCleanCacheRejournal,
		TrieCleanNoPrefetch: nodeConf.NoPrefetch,
		TrieDirtyLimit:      nodeConf.TrieDirtyCache,
		TrieDirtyDisabled:   nodeConf.NoPruning,
		TrieTimeLimit:       nodeConf.TrieTimeout,
		SnapshotLimit:       nodeConf.SnapshotCache,
		Preimages:           nodeConf.Preimages,
	}

	blockChain, err := core.NewBlockChain(chainDb, cacheConfig, chainConfig, engine, vmConfig, shouldPreserveFalse, &nodeConf.TxLookupLimit)
	if err != nil {
		t.Fatal(err)
	}

	currentState, err := blockChain.State()
	if err != nil {
		t.Fatal(err)
	}

	inboxwrapper := arbstate.NewInboxWrapper(currentState, blockChain.CurrentHeader(), blockChain)

	backend, err := arbitrum.NewBackend(stack, &nodeConf, chainDb, blockChain, arbos.ChainConfig.ChainID, inboxwrapper)
	if err != nil {
		t.Fatal(err)
	}

	apis := backend.APIBackend().GetAPIs()

	inproc := rpc.NewServer()
	for _, api := range apis {
		if err := inproc.RegisterName(api.Namespace, api.Service); err != nil {
			t.Fatal(err)
		}
	}

	client := ethclient.NewClient(rpc.DialInProc(inproc))

	return backend, client, ownerKey
}

//will wait untill tx is in the blockchain. attempts = 0 is infinite
func WaitForTx(t *testing.T, txhash common.Hash, backend *arbitrum.Backend, client *ethclient.Client, attempts int) {
	ctx := context.Background()
	chanHead := make(chan *types.Header, 20)
	headSubscribe, err := client.SubscribeNewHead(ctx, chanHead)
	if err != nil {
		t.Fatal(err)
	}
	defer headSubscribe.Unsubscribe()

	for {
		reciept, _ := client.TransactionReceipt(ctx, txhash)
		if reciept != nil {
			fmt.Println("Reciept: ", reciept)
			break
		}
		if attempts == 1 {
			t.Fatal("timeout waiting for Tx ", txhash)
		}
		if attempts > 1 {
			attempts -= 1
		}
		backend.CloseBlock()
		select {
		case <-chanHead:
		case <-time.After(time.Second / 100):
			backend.CloseBlock()
		}
	}
}

// TODO: is that right?
func shouldPreserveFalse(block *types.Block) bool {
	return false
}
