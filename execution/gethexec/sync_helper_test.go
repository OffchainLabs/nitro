package gethexec

import (
	"context"
	"errors"
	"math/big"
	"math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie"
)

type testBlockchainOptions struct {
	cachingConfig         *CachingConfig
	blocksNum             int
	forceTriedbCommitHook core.ForceTriedbCommitHook
}

func createTestBlockchain(t *testing.T, opts testBlockchainOptions) (*core.BlockChain, ethdb.Database) {
	if opts.cachingConfig == nil {
		opts.cachingConfig = &DefaultCachingConfig
	}
	stackConfig := node.DefaultConfig
	stackConfig.DataDir = t.TempDir()
	stackConfig.P2P.DiscoveryV4 = false
	stackConfig.P2P.DiscoveryV5 = false
	stackConfig.P2P.ListenAddr = "127.0.0.1:0"
	stack, err := node.New(&stackConfig)
	if err != nil {
		t.Fatal(err)
	}
	db, err := stack.OpenDatabaseWithFreezer("l2chaindata", 2048, 512, "", "", false)
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

	coreCacheConfig := DefaultCacheConfigFor(stack, opts.cachingConfig)
	bc, _ := core.NewArbBlockChain(db, coreCacheConfig, nil, gspec, nil, ethash.NewFaker(), vm.Config{}, nil, nil, opts.forceTriedbCommitHook)
	signer := types.MakeSigner(bc.Config(), big.NewInt(1), 0)

	_, blocks, allReceipts := core.GenerateChainWithGenesis(gspec, ethash.NewFaker(), opts.blocksNum, func(i int, gen *core.BlockGen) {
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
	return bc, db
}

type syncHelperScanTestOptions struct {
	blocksNum              int
	period                 int
	commitedCheckpointsNum int // 0 = all
}

func testSyncHelperScanNewConfirmedCheckpoints(t *testing.T, opts syncHelperScanTestOptions) {
	cachingConfig := DefaultCachingConfig
	cachingConfig.Archive = true
	cachingConfig.SnapshotCache = 0  // disable snapshot to simplify removing states
	cachingConfig.TrieCleanCache = 0 // disable trie/Database.cleans cache, so as states removed from ChainDb won't be cached there
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bc, db := createTestBlockchain(t, testBlockchainOptions{
		cachingConfig: &cachingConfig,
		blocksNum:     opts.blocksNum,
	})
	config := NitroSyncHelperConfig{
		Enabled:          true,
		CheckpointPeriod: uint64(opts.period),
		CheckpointCache:  uint(opts.blocksNum * 2), // big enough to detect bugs
	}
	sh := NewNitroSyncHelper(func() *NitroSyncHelperConfig { return &config }, bc)

	for number := 0; number < opts.blocksNum; number++ {
		header := bc.GetHeaderByNumber(uint64(number))
		if header == nil {
			t.Fatal("can't get header, number:", number, "opts:", opts)
		}
		if sh.checkpointCache.Has(header) {
			t.Fatal("unexpected error - checkpoint cache should be empty, but has header, number:", number, "opts:", opts)
		}
	}
	statesKept := make(map[int]struct{})
	if opts.commitedCheckpointsNum > 0 {
		toKeepCheckpoints := rand.Perm(opts.blocksNum / opts.period)[:opts.commitedCheckpointsNum]
		for _, checkpoint := range toKeepCheckpoints {
			block := (checkpoint + 1) * opts.period
			statesKept[block] = struct{}{}
		}
		for number := 1; number < opts.blocksNum; number++ {
			if _, keep := statesKept[number]; keep {
				continue
			}
			header := bc.GetHeaderByNumber(uint64(number))
			if header == nil {
				t.Fatal("can't get header, number:", number, "opts:", opts)
			}
			err := db.Delete(header.Root.Bytes())
			if err != nil {
				t.Fatal("failed to delete key from db, err:", err, "opts:", opts)
			}
			_, err = bc.StateAt(header.Root)
			if err == nil {
				t.Fatal("internal test error - failed to remove state from db", "number", number, "opts:", opts)
			}
			expectedErr := &trie.MissingNodeError{}
			if !errors.As(err, &expectedErr) {
				t.Fatal("internal test error - failed to remove state from db, err: ", err, "opts:", opts)
			}
		}
	}
	var previousConfirmed *Confirmed
	for number := 1; number < opts.blocksNum; number++ {
		block := bc.GetBlockByNumber(uint64(number))
		if block == nil {
			t.Fatal("can't get block, number:", number, "opts:", opts)
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
	for number := 0; number < opts.blocksNum; number++ {
		header := bc.GetHeaderByNumber(uint64(number))
		if header == nil {
			t.Fatal("can't get header, number:", number, "opts:", opts)
		}
		_, kept := statesKept[number]
		if number != 0 && number%opts.period == 0 && (opts.commitedCheckpointsNum == 0 || kept) {
			if !sh.checkpointCache.Has(header) {
				t.Fatal("checkpoint cache doesn't have expected header, number:", number, "opts:", opts)
			}
		} else if sh.checkpointCache.Has(header) {
			t.Fatal("checkpoint cache should not have the header, number:", number, "opts:", opts)
		}
	}
}

func TestSyncHelperScanNewConfirmedCheckpoints(t *testing.T) {
	options := []syncHelperScanTestOptions{}
	for i := 1; i < 7; i++ {
		options = append(options, syncHelperScanTestOptions{
			blocksNum:              51,
			period:                 i,
			commitedCheckpointsNum: 0,
		})
	}
	for i := 1; i < 7; i++ {
		options = append(options, syncHelperScanTestOptions{
			blocksNum:              51,
			period:                 4,
			commitedCheckpointsNum: rand.Intn(51/4) + 1,
		})
	}
	for _, o := range options {
		testSyncHelperScanNewConfirmedCheckpoints(t, o)
	}
}

func TestForceTriedbCommitHook(t *testing.T) {
	hook := GetForceTriedbCommitHookForConfig(func() *NitroSyncHelperConfig {
		return &NitroSyncHelperConfig{
			Enabled:          false,
			CheckpointPeriod: 100,
			CheckpointCache:  0,
		}
	})
	if hook != nil {
		t.Fatal("Got non nil hook, but NitroSyncHelper was disabled in the config")
	}
	hook = GetForceTriedbCommitHookForConfig(func() *NitroSyncHelperConfig {
		return &NitroSyncHelperConfig{
			Enabled:          true,
			CheckpointPeriod: 100,
			CheckpointCache:  0,
		}
	})
	for _, i := range []int64{1, 10, 99, 101} {
		header := &types.Header{
			Number: big.NewInt(i),
		}
		block := types.NewBlock(header, nil, nil, nil, nil)
		if hook(block, 0, 0) {
			t.Errorf("the hook returned true for block #%d", i)
		}
	}
	for _, i := range []int64{100, 200, 300} {
		header := &types.Header{
			Number: big.NewInt(i),
		}
		block := types.NewBlock(header, nil, nil, nil, nil)
		if !hook(block, 0, 0) {
			t.Errorf("the hook returned false for block #%d", i)
		}
	}
}
