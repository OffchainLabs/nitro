package mel

import (
	"context"
	"flag"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/offchainlabs/bold/solgen/go/rollupgen"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbstate/daprovider"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/staker/bold"
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

type MessageExtractionLayer struct {
	stopwaiter.StopWaiter
	config         MELConfigFetcher
	l1Reader       *headerreader.HeaderReader
	stateFetcher   StateFetcher
	addrs          *chaininfo.RollupAddresses
	melDB          StateDatabase
	sequencerInbox *arbnode.SequencerInbox
	dataProviders  []daprovider.Reader
}

func NewMessageExtractionLayer(
	l1Reader *headerreader.HeaderReader,
	rollupAddrs *chaininfo.RollupAddresses,
	stateFetcher StateFetcher,
	melDB StateDatabase,
	sequencerInbox *arbnode.SequencerInbox,
	dataProviders []daprovider.Reader,
	config MELConfigFetcher,
) (*MessageExtractionLayer, error) {
	if err := config().Validate(); err != nil {
		return nil, err
	}
	return &MessageExtractionLayer{
		l1Reader:       l1Reader,
		addrs:          rollupAddrs,
		stateFetcher:   stateFetcher,
		melDB:          melDB,
		sequencerInbox: sequencerInbox,
		dataProviders:  dataProviders,
		config:         config,
	}, nil
}

func (m *MessageExtractionLayer) Start(ctxIn context.Context) error {
	m.StopWaiter.Start(ctxIn, m)
	client := m.l1Reader.Client()
	rollup, err := rollupgen.NewRollupUserLogic(m.addrs.Rollup, client)
	if err != nil {
		return err
	}
	confirmedAssertionHash, err := rollup.LatestConfirmed(m.callOpts())
	if err != nil {
		return err
	}
	ctx := m.StopWaiter.GetContext()
	latestConfirmedAssertion, err := bold.ReadBoldAssertionCreationInfo(
		ctx,
		rollup,
		client,
		m.addrs.Rollup,
		confirmedAssertionHash,
	)
	if err != nil {
		return err
	}
	startBlock, err := client.HeaderByNumber(
		ctx,
		new(big.Int).SetUint64(latestConfirmedAssertion.CreationL1Block),
	)
	if err != nil {
		return err
	}
	state, err := m.stateFetcher.GetState(
		ctx,
		startBlock.Hash(),
	)
	if err != nil {
		return err
	}
	// TODO: Check this state parent chain id corresponds to
	// the node's configured chain id.
	for {
		latestBlock, err := client.HeaderByNumber(ctx, m.desiredBlockNumber())
		if err != nil {
			return err
		}
		endNum := latestBlock.Number.Uint64()
		postState, err := m.WalkForwards(
			ctx,
			state,
			client,
			endNum,
		)
		if err != nil {
			return err
		}
		state = postState
	}
}

type blockFetcher interface {
	BlockByNumber(
		ctx context.Context,
		number *big.Int,
	) (*types.Block, error)
	HeaderByNumber(
		ctx context.Context,
		number *big.Int,
	) (*types.Header, error)
}

func (m *MessageExtractionLayer) WalkForwards(
	ctx context.Context,
	initialState *State,
	blockFetcher blockFetcher,
	endBlockNumber uint64,
) (*State, error) {
	currNum := initialState.ParentChainBlockNumber
	state := initialState
	for currNum < endBlockNumber {
		parentChainBlock, err := blockFetcher.BlockByNumber(
			ctx,
			new(big.Int).SetUint64(currNum),
		)
		if err != nil {
			return nil, err
		}
		postState, msgs, err := m.extractMessages(
			ctx,
			state,
			parentChainBlock,
		)
		if err != nil {
			return nil, err
		}
		state = postState
		if err := m.melDB.SaveState(ctx, state, msgs); err != nil {
			return nil, err
		}
		currNum += 1
	}
	return state, nil
}

func (m *MessageExtractionLayer) desiredBlockNumber() *big.Int {
	readMode := m.config().ReadMode
	if readMode == "latest" {
		return big.NewInt(int64(rpc.LatestBlockNumber))
	} else if readMode == "safe" {
		return big.NewInt(int64(rpc.SafeBlockNumber))
	} else {
		return big.NewInt(int64(rpc.FinalizedBlockNumber))
	}
}

func (m *MessageExtractionLayer) callOpts() *bind.CallOpts {
	readMode := m.config().ReadMode
	if readMode == "latest" {
		return &bind.CallOpts{
			Context: m.StopWaiter.GetContext(),
		}
	} else if readMode == "safe" {
		return &bind.CallOpts{
			Context:     m.StopWaiter.GetContext(),
			BlockNumber: big.NewInt(int64(rpc.SafeBlockNumber)),
		}
	} else {
		return &bind.CallOpts{
			Context:     m.StopWaiter.GetContext(),
			BlockNumber: big.NewInt(int64(rpc.FinalizedBlockNumber)),
		}
	}
}
