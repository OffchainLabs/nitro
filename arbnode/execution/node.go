package execution

import (
	"errors"

	"github.com/ethereum/go-ethereum/arbitrum"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/eth/filters"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/node"
	"github.com/offchainlabs/nitro/util/headerreader"
)

type ExecutionNode struct {
	ChainDB      ethdb.Database
	Backend      *arbitrum.Backend
	FilterSystem *filters.FilterSystem
	ArbInterface *ArbInterface
	ExecEngine   *ExecutionEngine
	Sequencer    *Sequencer // either nil or same as TxPublisher
	TxPublisher  TransactionPublisher
}

func CreateExecutionNode(
	stack *node.Node,
	chainDB ethdb.Database,
	l2BlockChain *core.BlockChain,
	l1Reader *headerreader.HeaderReader,
	syncMonitor arbitrum.SyncProgressBackend,
	fwTarget string,
	fwConfig *ForwarderConfig,
	rpcConfig arbitrum.Config,
	seqConfigFetcher SequencerConfigFetcher,
	strictnessFetcher func() uint,
) (*ExecutionNode, error) {
	execEngine, err := NewExecutionEngine(l2BlockChain)
	if err != nil {
		return nil, err
	}
	var txPublisher TransactionPublisher
	var sequencer *Sequencer
	seqConfig := seqConfigFetcher()
	if seqConfig.Enable {
		if fwTarget != "" {
			return nil, errors.New("sequencer and forwarding target both set")
		}
		sequencer, err = NewSequencer(execEngine, l1Reader, seqConfigFetcher)
		if err != nil {
			return nil, err
		}
		txPublisher = sequencer
	} else {
		if fwConfig.RedisUrl != "" {
			txPublisher = NewRedisTxForwarder(fwTarget, fwConfig)
		} else if fwTarget == "" {
			txPublisher = NewTxDropper()
		} else {
			txPublisher = NewForwarder(fwTarget, fwConfig)
		}
	}

	txPublisher = NewTxPreChecker(txPublisher, l2BlockChain, strictnessFetcher)
	arbInterface, err := NewArbInterface(execEngine, txPublisher)
	if err != nil {
		return nil, err
	}
	filterConfig := filters.Config{
		LogCacheSize: rpcConfig.FilterLogCacheSize,
		Timeout:      rpcConfig.FilterTimeout,
	}
	backend, filterSystem, err := arbitrum.NewBackend(stack, &rpcConfig, chainDB, arbInterface, syncMonitor, filterConfig)
	if err != nil {
		return nil, err
	}

	return &ExecutionNode{
		chainDB,
		backend,
		filterSystem,
		arbInterface,
		execEngine,
		sequencer,
		txPublisher,
	}, nil

}
