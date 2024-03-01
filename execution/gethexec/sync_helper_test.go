package gethexec

import (
	"context"
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
)

func createTestBlockchain(t *testing.T, blocksNum int) *core.BlockChain {
	stackConfig := node.DefaultConfig
	stackConfig.DataDir = t.TempDir()
	stackConfig.P2P.DiscoveryV4 = false
	stackConfig.P2P.DiscoveryV5 = false
	stackConfig.P2P.ListenAddr = "127.0.0.1:0"
	stack, err := node.New(&stackConfig)
	if err != nil {
		t.Fatal(err)
	}
	chaindb, err := stack.OpenDatabaseWithFreezer("l2chaindata", 2048, 512, "", "", false)
	if err != nil {
		t.Fatal(err)
	}

	// create and populate chain
	testUser, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal("generate key err:", err)
	}
	testUserAddress := crypto.PubkeyToAddress(testUser.PublicKey)

	gspec := &core.Genesis{
		Config: params.TestChainConfig,
		Alloc: core.GenesisAlloc{
			testUserAddress: {Balance: new(big.Int).Lsh(big.NewInt(1), 250)},
		},
	}
	cachingConfig := DefaultCachingConfig
	cachingConfig.Archive = true
	coreCacheConfig := DefaultCacheConfigFor(stack, &cachingConfig)
	bc, _ := core.NewArbBlockChain(chaindb, coreCacheConfig, nil, gspec, nil, ethash.NewFaker(), vm.Config{}, nil, nil, nil)
	signer := types.MakeSigner(bc.Config(), big.NewInt(1), 0)

	_, blocks, allReceipts := core.GenerateChainWithGenesis(gspec, ethash.NewFaker(), blocksNum, func(i int, gen *core.BlockGen) {
		nonce := gen.TxNonce(testUserAddress)
		tx, err := types.SignNewTx(testUser, signer, &types.LegacyTx{
			Nonce:    nonce,
			GasPrice: gen.BaseFee(),
			Gas:      uint64(1000001),
		})
		if err != nil {
			t.Fatalf("failed to create tx: %v", err)
		}
		gen.AddTx(tx)

	})
	for _, receipts := range allReceipts {
		if len(receipts) < 1 {
			t.Fatal("missing receipts")
		}
		for _, receipt := range receipts {
			if receipt.Status == 0 {
				t.Fatal("failed transaction")
			}
		}
	}
	if _, err := bc.InsertChain(blocks); err != nil {
		t.Fatal(err)
	}
	return bc
}

func TestSyncHelperScanNewConfirmedCheckpointsAllAvailable(t *testing.T) {
	period := 7
	blocksNum := 100
	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(false)))
	glogger.Verbosity(log.LvlInfo)
	log.Root().SetHandler(glogger)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bc := createTestBlockchain(t, blocksNum)
	config := NitroSyncHelperConfig{
		Enabled:          true,
		CheckpointPeriod: uint64(period),
		CheckpointCache:  uint(blocksNum * 2), // big enough to detect bugs
	}
	sh := NewNitroSyncHelper(func() *NitroSyncHelperConfig { return &config }, bc)

	for number := 0; number < blocksNum; number++ {
		header := bc.GetHeaderByNumber(uint64(number))
		if header == nil {
			t.Fatal("internal test error - can't get header, number:", number)
		}
		if sh.checkpointCache.Has(header) {
			t.Fatal("unexpected error - checkpoint cache should be empty, but has header, number:", number)
		}
	}
	var previousConfirmed *Confirmed
	for number := 1; number < blocksNum; number++ {
		block := bc.GetBlockByNumber(uint64(number))
		if block == nil {
			t.Fatal("internal test error - can't get block, number:", number)
		}
		newConfirmed := Confirmed{
			BlockNumber: int64(number),
			BlockHash:   block.Hash(),
			Node:        0, // doesn't metter here
			Header:      block.Header(),
		}
		sh.scanNewConfirmedCheckpoints(ctx, &newConfirmed, previousConfirmed)
		previousConfirmed = &newConfirmed
	}
	for number := 0; number < blocksNum; number++ {
		header := bc.GetHeaderByNumber(uint64(number))
		if header == nil {
			t.Fatal("internal test error - can't get header, number:", number)
		}
		if number != 0 && number%period == 0 {
			if !sh.checkpointCache.Has(header) {
				t.Fatal("checkpoint cache doesn't have expected header, number:", number)
			}
		} else if sh.checkpointCache.Has(header) {
			t.Fatal("checkpoint cache should not have the header, number:", number)
		}
	}

}
