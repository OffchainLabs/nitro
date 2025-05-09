package mel

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/bold/containers/fsm"
	meltypes "github.com/offchainlabs/nitro/arbnode/message-extraction/types"
	"github.com/offchainlabs/nitro/arbstate/daprovider"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

type MessageExtractor struct {
	stopwaiter.StopWaiter
	l1Reader                  *headerreader.HeaderReader
	stateFetcher              meltypes.StateFetcher
	addrs                     *chaininfo.RollupAddresses
	melDB                     meltypes.StateDatabase
	dataProviders             []daprovider.Reader
	startParentChainBlockHash common.Hash
	fsm                       *fsm.Fsm[action, FSMState]
}

func NewMessageExtractor(
	l1Reader *headerreader.HeaderReader,
	rollupAddrs *chaininfo.RollupAddresses,
	stateFetcher meltypes.StateFetcher,
	melDB meltypes.StateDatabase,
	dataProviders []daprovider.Reader,
	startParentChainBlockHash common.Hash,
) (*MessageExtractor, error) {
	fsm, err := newFSM(Start)
	if err != nil {
		return nil, err
	}
	return &MessageExtractor{
		l1Reader:                  l1Reader,
		addrs:                     rollupAddrs,
		stateFetcher:              stateFetcher,
		melDB:                     melDB,
		dataProviders:             dataProviders,
		startParentChainBlockHash: startParentChainBlockHash,
		fsm:                       fsm,
	}, nil
}

func (m *MessageExtractor) Start(ctxIn context.Context) error {
	m.StopWaiter.Start(ctxIn, m)
	runChan := make(chan struct{}, 1)
	return stopwaiter.CallIterativelyWith(
		&m.StopWaiterSafe,
		func(ctx context.Context, ignored struct{}) time.Duration {
			actAgainInterval, err := m.Act(ctx)
			if err != nil {
				log.Error("Error in message extractor", "err", err)
			}
			return actAgainInterval
		},
		runChan,
	)
}
