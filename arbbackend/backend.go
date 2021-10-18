package arbbackend

import (
	"bytes"
	"math/big"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/arbstate/arbos"
)

type ArbBackend struct {
	segmentQueue []*arbos.MessageSegment
	blockChain   *core.BlockChain
	stack        *node.Node
	chainId      *big.Int
	apiBackend   *ArbAPIBackend
	ethConfig    *ethconfig.Config
	ethDatabase  ethdb.Database

	chanSegments chan *arbos.MessageSegment
	chanClose    chan struct{} //close coroutine
	chanNewBlock chan struct{} //create new L2 block unless empty
}

func New(stack *node.Node, config *ethconfig.Config) (*ArbBackend, error) {
	chainDb, err := stack.OpenDatabaseWithFreezer("chaindata", config.DatabaseCache, config.DatabaseHandles, config.DatabaseFreezer, "eth/db/chaindata/", false)
	if err != nil {
		return nil, err
	}
	engine := arbos.Engine{
		IsSequencer: true,
	}
	chainConfig, _, genesisErr := core.SetupGenesisBlockWithOverride(chainDb, config.Genesis, config.OverrideLondon)
	if _, ok := genesisErr.(*params.ConfigCompatError); genesisErr != nil && !ok {
		return nil, genesisErr
	}

	vmConfig := vm.Config{
		EnablePreimageRecording: config.EnablePreimageRecording,
	}
	cacheConfig := &core.CacheConfig{
		TrieCleanLimit:      config.TrieCleanCache,
		TrieCleanJournal:    stack.ResolvePath(config.TrieCleanCacheJournal),
		TrieCleanRejournal:  config.TrieCleanCacheRejournal,
		TrieCleanNoPrefetch: config.NoPrefetch,
		TrieDirtyLimit:      config.TrieDirtyCache,
		TrieDirtyDisabled:   config.NoPruning,
		TrieTimeLimit:       config.TrieTimeout,
		SnapshotLimit:       config.SnapshotCache,
		Preimages:           config.Preimages,
	}

	blockChain, err := core.NewBlockChain(chainDb, cacheConfig, chainConfig, engine, vmConfig, shouldPreserveFalse, &config.TxLookupLimit)
	if err != nil {
		return nil, err
	}
	backend := &ArbBackend{
		segmentQueue: make([]*arbos.MessageSegment, 0),
		blockChain:   blockChain,
		stack:        stack,
		chainId:      arbos.ChainConfig.ChainID,
		ethConfig:    config,
		ethDatabase:  chainDb,
		chanSegments: make(chan *arbos.MessageSegment, 100),
		chanClose:    make(chan struct{}, 1),
		chanNewBlock: make(chan struct{}, 1),
	}
	stack.RegisterLifecycle(backend)
	go backend.segmentQueueRutine()

	createRegisterAPIBackend(backend)
	return backend, nil
}

func (b *ArbBackend) EnqueueL2Message(tx *types.Transaction) error {
	l1msgKind_l2Msg := []byte{arbos.L1MessageType_L2Message}
	l1msgFields_tmp := make([]byte, 32*5) //TODO: all fields currently zeroed
	var buf bytes.Buffer
	err := tx.EncodeRLP(&buf)
	if err != nil {
		return err
	}
	l2msgKind_signedTx := []byte{arbos.L2MessageKind_SignedTx}
	l2msg := append(l2msgKind_signedTx, buf.Bytes()...)
	l1msg := append(l1msgKind_l2Msg, append(l1msgFields_tmp, l2msg...)...)
	newSegments, err := arbos.ParseIncomingL1Message(bytes.NewReader(l1msg), b.chainId)
	if err != nil {
		return err
	}
	for _, seg := range newSegments {
		b.chanSegments <- seg
	}
	return nil
}

func (b *ArbBackend) CloseBlock() {
	b.chanNewBlock <- struct{}{}
}

func (b *ArbBackend) tryBuildBlock(force bool) (bool, error) {
	if len(b.segmentQueue) == 0 {
		return false, nil
	}
	currentState, err := b.blockChain.State()
	if err != nil {
		return false, err
	}
	blockBuilder := arbos.NewBlockBuilder(currentState, b.blockChain.CurrentHeader(), b.blockChain)
	segmentDone := true
	iSegment := 0
	// TODO: understand blockbuilder better
	for segmentDone && (iSegment < len(b.segmentQueue)) {
		segmentDone = blockBuilder.AddSegment(b.segmentQueue[iSegment])
		iSegment += 1
	}
	if force {
		blockBuilder.ForceCloseBlock()
	}
	block, state, reciepts := blockBuilder.PendingBlock()
	if block == nil {
		return false, nil
	}
	logs := make([]*types.Log, 0)
	for _, receipt := range reciepts {
		logs = append(logs, receipt.Logs...)
	}
	b.blockChain.WriteBlockWithState(block, reciepts, logs, state, true)
	b.segmentQueue = b.segmentQueue[iSegment-1:]
	return true, nil
}

func (b *ArbBackend) segmentQueueRutine() {
	for {
		select {
		case segment := <-b.chanSegments:
			b.segmentQueue = append(b.segmentQueue, segment)
			tryBuilding := true
			for tryBuilding {
				var err error
				tryBuilding, err = b.tryBuildBlock(false)
				if err != nil {
					panic(err)
				}
			}
		case <-b.chanNewBlock:
			tryBuilding := true
			for tryBuilding {
				var err error
				tryBuilding, err = b.tryBuildBlock(true)
				if err != nil {
					panic(err)
				}
			}
		case <-b.chanClose:
			return
		}
	}
}

//TODO: this is used when registering backend as lifecycle in stack
func (b *ArbBackend) Start() error {
	return nil
}

func (b *ArbBackend) Stop() error {

	b.blockChain.Stop()

	return nil
}

// TODO: is that right?
func shouldPreserveFalse(block *types.Block) bool {
	return false
}
