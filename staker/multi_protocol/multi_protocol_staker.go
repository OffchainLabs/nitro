package multiprotocolstaker

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"

	"github.com/offchainlabs/bold/solgen/go/bridgegen"
	boldrollup "github.com/offchainlabs/bold/solgen/go/rollupgen"
	"github.com/offchainlabs/nitro/staker"
	boldstaker "github.com/offchainlabs/nitro/staker/bold"
	legacystaker "github.com/offchainlabs/nitro/staker/legacy"
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
	boldStaker              *boldstaker.BOLDStaker
	legacyConfig            legacystaker.L1ValidatorConfigFetcher
	stakedNotifiers         []legacystaker.LatestStakedNotifier
	confirmedNotifiers      []legacystaker.LatestConfirmedNotifier
	statelessBlockValidator *staker.StatelessBlockValidator
	wallet                  legacystaker.ValidatorWalletInterface
	l1Reader                *headerreader.HeaderReader
	blockValidator          *staker.BlockValidator
	callOpts                bind.CallOpts
	boldConfig              *boldstaker.BoldConfig
	stakeTokenAddress       common.Address
	stack                   *node.Node
}

func NewMultiProtocolStaker(
	stack *node.Node,
	l1Reader *headerreader.HeaderReader,
	wallet legacystaker.ValidatorWalletInterface,
	callOpts bind.CallOpts,
	legacyConfig legacystaker.L1ValidatorConfigFetcher,
	boldConfig *boldstaker.BoldConfig,
	blockValidator *staker.BlockValidator,
	statelessBlockValidator *staker.StatelessBlockValidator,
	stakedNotifiers []legacystaker.LatestStakedNotifier,
	stakeTokenAddress common.Address,
	confirmedNotifiers []legacystaker.LatestConfirmedNotifier,
	validatorUtilsAddress common.Address,
	bridgeAddress common.Address,
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
	}, nil
}

func (m *MultiProtocolStaker) Initialize(ctx context.Context) error {
	boldActive, rollupAddress, err := m.isBoldActive(ctx)
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
		m.CallIteratively(func(ctx context.Context) time.Duration {
			switchedToBoldProtocol, err := m.checkAndSwitchToBoldStaker(ctxIn)
			if err != nil {
				log.Warn("staker: error in checking switch to bold staker", "err", err)
				return stakerSwitchInterval
			}
			if switchedToBoldProtocol {
				log.Info("Detected BOLD protocol upgrade, stopping old staker and starting BOLD staker")
				// Ready to stop the old staker.
				m.oldStaker.StopOnly()
				m.StopOnly()
			}
			return stakerSwitchInterval
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

func (m *MultiProtocolStaker) isBoldActive(ctx context.Context) (bool, common.Address, error) {
	var addr common.Address
	if !m.boldConfig.Enable {
		return false, addr, nil
	}
	callOpts := m.getCallOpts(ctx)
	rollupAddress, err := m.bridge.Rollup(callOpts)
	if err != nil {
		return false, addr, err
	}
	userLogic, err := boldrollup.NewRollupUserLogic(rollupAddress, m.l1Reader.Client())
	if err != nil {
		return false, addr, err
	}
	_, err = userLogic.ChallengeGracePeriodBlocks(callOpts)
	if err != nil && !headerreader.ExecutionRevertedRegexp.MatchString(err.Error()) {
		// Unexpected error, perhaps an L1 issue?
		return false, addr, err
	}
	// ChallengeGracePeriodBlocks only exists in the BOLD rollup contracts.
	return err == nil, rollupAddress, nil
}

func (m *MultiProtocolStaker) checkAndSwitchToBoldStaker(ctx context.Context) (bool, error) {
	shouldSwitch, rollupAddress, err := m.isBoldActive(ctx)
	if err != nil {
		return false, err
	}
	if !shouldSwitch {
		return false, nil
	}
	if err := m.setupBoldStaker(ctx, rollupAddress); err != nil {
		return false, err
	}
	if err = m.boldStaker.Initialize(ctx); err != nil {
		return false, err
	}
	m.boldStaker.Start(ctx)
	return true, nil
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
	boldStaker, err := boldstaker.NewBOLDStaker(
		ctx,
		m.stack,
		rollupAddress,
		m.callOpts,
		txBuilder.SingleTxAuth(),
		m.l1Reader,
		m.blockValidator,
		m.statelessBlockValidator,
		m.boldConfig,
		m.wallet.DataPoster(),
		m.wallet,
		m.stakedNotifiers,
		m.confirmedNotifiers,
	)
	if err != nil {
		return err
	}
	m.boldStaker = boldStaker
	return nil
}
