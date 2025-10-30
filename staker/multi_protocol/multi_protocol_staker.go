package multiprotocolstaker

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"

	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/solgen/go/rollupgen"
	"github.com/offchainlabs/nitro/staker"
	"github.com/offchainlabs/nitro/staker/bold"
	"github.com/offchainlabs/nitro/staker/legacy"
	"github.com/offchainlabs/nitro/staker/txbuilder"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

const boldArt = `
 _______             __        _______
/       \           /  |      /       \
$$$$$$$  |  ______  $$ |      $$$$$$$  |
$$ |__$$ | /      \ $$ |      $$ |  $$ |
$$    $$< /$$$$$$  |$$ |      $$ |  $$ |
$$$$$$$  |$$ |  $$ |$$ |      $$ |  $$ |
$$ |__$$ |$$ \__$$ |$$ |_____ $$ |__$$ |
$$    $$/ $$    $$/ $$       |$$    $$/
$$$$$$$/   $$$$$$/  $$$$$$$$/ $$$$$$$/
`

type MultiProtocolStaker struct {
	stopwaiter.StopWaiter
	bridge                  *bridgegen.IBridge
	oldStaker               *legacystaker.Staker
	boldStaker              *bold.BOLDStaker
	legacyConfig            legacystaker.L1ValidatorConfigFetcher
	stakedNotifiers         []legacystaker.LatestStakedNotifier
	confirmedNotifiers      []legacystaker.LatestConfirmedNotifier
	statelessBlockValidator *staker.StatelessBlockValidator
	wallet                  legacystaker.ValidatorWalletInterface
	l1Reader                *headerreader.HeaderReader
	blockValidator          *staker.BlockValidator
	callOpts                bind.CallOpts
	boldConfig              *bold.BoldConfig
	stakeTokenAddress       common.Address
	stack                   *node.Node
	inboxTracker            staker.InboxTrackerInterface
	inboxStreamer           staker.TransactionStreamerInterface
	inboxReader             staker.InboxReaderInterface
	dapRegistry             *daprovider.DAProviderRegistry
	fatalErr                chan<- error
}

func NewMultiProtocolStaker(
	stack *node.Node,
	l1Reader *headerreader.HeaderReader,
	wallet legacystaker.ValidatorWalletInterface,
	callOpts bind.CallOpts,
	legacyConfig legacystaker.L1ValidatorConfigFetcher,
	boldConfig *bold.BoldConfig,
	blockValidator *staker.BlockValidator,
	statelessBlockValidator *staker.StatelessBlockValidator,
	stakedNotifiers []legacystaker.LatestStakedNotifier,
	stakeTokenAddress common.Address,
	rollupAddress common.Address,
	confirmedNotifiers []legacystaker.LatestConfirmedNotifier,
	validatorUtilsAddress common.Address,
	bridgeAddress common.Address,
	inboxStreamer staker.TransactionStreamerInterface,
	inboxTracker staker.InboxTrackerInterface,
	inboxReader staker.InboxReaderInterface,
	dapRegistry *daprovider.DAProviderRegistry,
	fatalErr chan<- error,
) (*MultiProtocolStaker, error) {
	if err := legacyConfig().Validate(); err != nil {
		return nil, err
	}
	if legacyConfig().StartValidationFromStaked && blockValidator != nil {
		stakedNotifiers = append(stakedNotifiers, blockValidator)
	}
	oldStaker, err := legacystaker.NewStaker(
		l1Reader,
		wallet,
		callOpts,
		legacyConfig,
		blockValidator,
		statelessBlockValidator,
		stakedNotifiers,
		confirmedNotifiers,
		validatorUtilsAddress,
		rollupAddress,
		inboxTracker,
		inboxStreamer,
		inboxReader,
		fatalErr,
	)
	if err != nil {
		return nil, err
	}
	bridge, err := bridgegen.NewIBridge(bridgeAddress, l1Reader.Client())
	if err != nil {
		return nil, err
	}
	return &MultiProtocolStaker{
		oldStaker:               oldStaker,
		boldStaker:              nil,
		bridge:                  bridge,
		legacyConfig:            legacyConfig,
		stakedNotifiers:         stakedNotifiers,
		confirmedNotifiers:      confirmedNotifiers,
		statelessBlockValidator: statelessBlockValidator,
		wallet:                  wallet,
		l1Reader:                l1Reader,
		blockValidator:          blockValidator,
		callOpts:                callOpts,
		boldConfig:              boldConfig,
		stakeTokenAddress:       stakeTokenAddress,
		stack:                   stack,
		inboxTracker:            inboxTracker,
		inboxStreamer:           inboxStreamer,
		inboxReader:             inboxReader,
		dapRegistry:             dapRegistry,
		fatalErr:                fatalErr,
	}, nil
}

func (m *MultiProtocolStaker) Initialize(ctx context.Context) error {
	boldActive, rollupAddress, err := IsBoldActive(m.getCallOpts(ctx), m.bridge, m.l1Reader.Client())
	if err != nil {
		return err
	}
	if boldActive {
		log.Info("BoLD protocol is active, initializing BoLD staker")
		log.Info(boldArt)
		if err := m.setupBoldStaker(ctx, rollupAddress); err != nil {
			return err
		}
		m.oldStaker = nil
		return m.boldStaker.Initialize(ctx)
	}
	log.Info("BoLD protocol not detected on startup, using old staker until upgrade")
	return m.oldStaker.Initialize(ctx)
}

func (m *MultiProtocolStaker) Start(ctxIn context.Context) {
	m.StopWaiter.Start(ctxIn, m)
	m.wallet.Start(ctxIn)
	if m.boldStaker != nil {
		log.Info("Starting BOLD staker")
		m.boldStaker.Start(ctxIn)
	} else {
		log.Info("Starting pre-BOLD staker")
		m.oldStaker.Start(ctxIn)
		stakerSwitchInterval := m.boldConfig.CheckStakerSwitchInterval
		m.LaunchThread(func(ctx context.Context) {
			ticker := time.NewTicker(stakerSwitchInterval)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
				}

				if err := m.checkAndSwitchToBoldStaker(ctxIn); err != nil {
					log.Warn("Staker: error in checking and switching to bold staker", "err", err)
					continue
				}
				return
			}
		})
	}
}

func (m *MultiProtocolStaker) StopAndWait() {
	if m.boldStaker != nil {
		m.boldStaker.StopAndWait()
	}
	if m.oldStaker != nil {
		m.oldStaker.StopAndWait()
	}
	m.StopWaiter.StopAndWait()
}

func IsBoldActive(callOpts *bind.CallOpts, bridge *bridgegen.IBridge, l1Backend *ethclient.Client) (bool, common.Address, error) {
	var addr common.Address
	rollupAddress, err := bridge.Rollup(callOpts)
	if err != nil {
		return false, addr, err
	}
	userLogic, err := rollupgen.NewRollupUserLogic(rollupAddress, l1Backend)
	if err != nil {
		return false, addr, err
	}
	_, err = userLogic.ChallengeGracePeriodBlocks(callOpts)
	if err != nil && !headerreader.IsExecutionReverted(err) {
		// Unexpected error, perhaps an L1 issue?
		return false, addr, err
	}
	// ChallengeGracePeriodBlocks only exists in the BOLD rollup contracts.
	return err == nil, rollupAddress, nil
}

func (m *MultiProtocolStaker) checkAndSwitchToBoldStaker(ctx context.Context) error {
	shouldSwitch, rollupAddress, err := IsBoldActive(m.getCallOpts(ctx), m.bridge, m.l1Reader.Client())
	if err != nil {
		return err
	}
	if !shouldSwitch {
		log.Info("Bold is not yet active on-chain, will retry switching later")
		return nil
	}
	if err := m.setupBoldStaker(ctx, rollupAddress); err != nil {
		return err
	}
	if err = m.boldStaker.Initialize(ctx); err != nil {
		return err
	}
	log.Info("Detected BOLD protocol upgrade, stopping old staker and starting BOLD staker")
	m.boldStaker.Start(ctx)
	// Ready to stop the old staker.
	m.oldStaker.StopOnly()
	m.StopOnly()
	return nil
}

func (m *MultiProtocolStaker) getCallOpts(ctx context.Context) *bind.CallOpts {
	opts := m.callOpts
	opts.Context = ctx
	return &opts
}

func (m *MultiProtocolStaker) setupBoldStaker(
	ctx context.Context,
	rollupAddress common.Address,
) error {
	stakeTokenContract, err := m.l1Reader.Client().CodeAt(ctx, m.stakeTokenAddress, nil)
	if err != nil {
		return err
	}
	if len(stakeTokenContract) == 0 {
		return fmt.Errorf("stake token address for BoLD %v does not point to a contract", m.stakeTokenAddress)
	}
	txBuilder, err := txbuilder.NewBuilder(m.wallet, m.legacyConfig().GasRefunder())
	if err != nil {
		return err
	}
	boldStaker, err := bold.NewBOLDStaker(
		ctx,
		m.stack,
		rollupAddress,
		m.callOpts,
		txBuilder.SingleTxAuth(),
		m.l1Reader,
		m.blockValidator,
		m.statelessBlockValidator,
		m.boldConfig,
		m.legacyConfig().StrategyType(),
		m.wallet.DataPoster(),
		m.wallet,
		m.stakedNotifiers,
		m.confirmedNotifiers,
		m.inboxTracker,
		m.inboxStreamer,
		m.inboxReader,
		m.dapRegistry,
		m.fatalErr,
	)
	if err != nil {
		return err
	}
	m.boldStaker = boldStaker
	return nil
}
