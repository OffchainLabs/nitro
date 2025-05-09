package mel

import (
	"context"
	"flag"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/bold/containers/fsm"
	meltypes "github.com/offchainlabs/nitro/arbnode/message-extraction/types"
	"github.com/offchainlabs/nitro/arbstate/daprovider"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

func MELConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.String(prefix+".read-mode", DefaultMELConfig.ReadMode, "mode to only read latest or safe or finalized L1 blocks. Enabling safe or finalized disables feed input and output. Defaults to latest. Takes string input, valid strings- latest, safe, finalized")
}

type MELConfig struct {
	ReadMode string `koanf:"read-mode" reload:"hot"`
}

type MELConfigFetcher func() *MELConfig

func (c *MELConfig) Validate() error {
	c.ReadMode = strings.ToLower(c.ReadMode)
	if c.ReadMode != "latest" && c.ReadMode != "safe" && c.ReadMode != "finalized" {
		return fmt.Errorf("inbox reader read-mode is invalid, want: latest or safe or finalized, got: %s", c.ReadMode)
	}
	return nil
}

var DefaultMELConfig = MELConfig{
	ReadMode: "latest",
}

var TestMELConfig = MELConfig{
	ReadMode: "latest",
}

type MessageExtractor struct {
	stopwaiter.StopWaiter
	config        MELConfigFetcher
	l1Reader      *headerreader.HeaderReader
	stateFetcher  meltypes.StateFetcher
	addrs         *chaininfo.RollupAddresses
	melDB         meltypes.StateDatabase
	dataProviders []daprovider.Reader
	fsm           *fsm.Fsm[action, FSMState]
}

func NewMessageExtractor(
	l1Reader *headerreader.HeaderReader,
	rollupAddrs *chaininfo.RollupAddresses,
	stateFetcher meltypes.StateFetcher,
	melDB meltypes.StateDatabase,
	dataProviders []daprovider.Reader,
	config MELConfigFetcher,
) (*MessageExtractor, error) {
	if err := config().Validate(); err != nil {
		return nil, err
	}
	fsm, err := newFSM(Start)
	if err != nil {
		return nil, err
	}
	return &MessageExtractor{
		l1Reader:      l1Reader,
		addrs:         rollupAddrs,
		stateFetcher:  stateFetcher,
		melDB:         melDB,
		dataProviders: dataProviders,
		config:        config,
		fsm:           fsm,
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
				return actAgainInterval
			}
			return actAgainInterval
		},
		runChan,
	)
}

func (m *MessageExtractor) callOpts(ctx context.Context) *bind.CallOpts {
	readMode := m.config().ReadMode
	if readMode == "latest" {
		return &bind.CallOpts{
			Context: ctx,
		}
	} else if readMode == "safe" {
		return &bind.CallOpts{
			Context:     ctx,
			BlockNumber: big.NewInt(int64(rpc.SafeBlockNumber)),
		}
	} else {
		return &bind.CallOpts{
			Context:     ctx,
			BlockNumber: big.NewInt(int64(rpc.FinalizedBlockNumber)),
		}
	}
}
