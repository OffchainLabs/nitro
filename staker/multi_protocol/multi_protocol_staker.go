// Copyright 2024-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
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
	// wallet is started externally (with the raw ctxIn so it outlives StopOnly during
	// protocol switches) but owned and stopped by MultiProtocolStaker.StopAndWait.
	wallet legacystaker.ValidatorWalletInterface
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
	// Wallet is started with the external context because it must outlive
	// a potential old→bold staker switch (which calls m.StopOnly).
	// It is NOT tracked via TrackChild — its lifecycle is managed explicitly in StopAndWait.
	m.wallet.Start(ctxIn)
	if m.boldStaker != nil {
		log.Info("Starting BOLD staker")
		m.StartAndTrackChild(m.boldStaker)
	} else {
		log.Info("Starting pre-BOLD staker")
		m.StartAndTrackChild(m.oldStaker)
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
	// boldStaker is intentionally NOT tracked as a child: it must outlive the StopOnly call
	// below (which cancels m's managed context and stops tracked children like oldStaker).
	// StopAndWait will stop it explicitly after all goroutines have exited.
	m.boldStaker.Start(ctx)
	// Cancel m's managed context and stop tracked children (i.e. oldStaker).
	// After this call the calling goroutine's context is also cancelled, so the
	// goroutine must return promptly to allow wg.Wait() to complete.
	m.StopOnly()
	return nil
}

func (m *MultiProtocolStaker) StopAndWait() {
	// oldStaker may have been started dynamically and stopped via StopOnly (TrackChild),
	// but its goroutines still need waiting. Explicit StopAndWait is idempotent if it
	// was already fully stopped.
	if m.oldStaker != nil {
		m.oldStaker.StopAndWait()
	}
	// Wait for m's own goroutines (including the potential switch goroutine) to exit.
	// This must happen before reading m.boldStaker: the switch goroutine writes
	// m.boldStaker and calling StopWaiter.StopAndWait() guarantees it has exited,
	// making the subsequent read below race-free without requiring a mutex.
	m.StopWaiter.StopAndWait()
	// boldStaker is not tracked (see checkAndSwitchToBoldStaker), so stop it explicitly.
	// Safe to read m.boldStaker here because the switch goroutine has already exited.
	if m.boldStaker != nil {
		m.boldStaker.StopAndWait()
	}
	// Wallet is started with external context, so stop it last.
	m.wallet.StopAndWait()
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
