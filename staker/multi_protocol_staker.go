package staker

import (
	"context"
	"time"

	"github.com/OffchainLabs/bold/solgen/go/bridgegen"
	boldrollup "github.com/OffchainLabs/bold/solgen/go/rollupgen"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	oldrollupgen "github.com/offchainlabs/nitro/solgen/go/rollupgen"
	"github.com/offchainlabs/nitro/staker/txbuilder"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

var assertionCreatedId common.Hash

func init() {
	rollupAbi, err := boldrollup.RollupCoreMetaData.GetAbi()
	if err != nil {
		panic(err)
	}
	assertionCreatedEvent, ok := rollupAbi.Events["AssertionCreated"]
	if !ok {
		panic("RollupCore ABI missing AssertionCreated event")
	}
	assertionCreatedId = assertionCreatedEvent.ID
}

type MultiProtocolStaker struct {
	stopwaiter.StopWaiter
	bridge     *bridgegen.IBridge
	oldStaker  *Staker
	boldStaker *BOLDStaker
}

func NewMultiProtocolStaker(
	l1Reader *headerreader.HeaderReader,
	wallet ValidatorWalletInterface,
	callOpts bind.CallOpts,
	config L1ValidatorConfig,
	blockValidator *BlockValidator,
	statelessBlockValidator *StatelessBlockValidator,
	stakedNotifiers []LatestStakedNotifier,
	confirmedNotifiers []LatestConfirmedNotifier,
	validatorUtilsAddress common.Address,
	bridgeAddress common.Address,
	fatalErr chan<- error,
) (*MultiProtocolStaker, error) {
	oldStaker, err := NewStaker(
		l1Reader,
		wallet,
		callOpts,
		config,
		blockValidator,
		statelessBlockValidator,
		stakedNotifiers,
		confirmedNotifiers,
		validatorUtilsAddress,
		bridgeAddress,
		fatalErr,
	)
	if err != nil {
		return nil, err
	}
	bridge, err := bridgegen.NewIBridge(bridgeAddress, oldStaker.client)
	if err != nil {
		return nil, err
	}
	return &MultiProtocolStaker{
		oldStaker:  oldStaker,
		boldStaker: nil,
		bridge:     bridge,
	}, nil
}

func (m *MultiProtocolStaker) IsWhitelisted(ctx context.Context) (bool, error) {
	return false, nil
}

func (m *MultiProtocolStaker) Initialize(ctx context.Context) error {
	boldActive, _, err := m.isBoldActive(ctx)
	if err != nil {
		return err
	}
	if boldActive {
		txBuilder, err := txbuilder.NewBuilder(m.oldStaker.wallet)
		if err != nil {
			return err
		}
		auth, err := txBuilder.Auth(ctx)
		if err != nil {
			return err
		}
		boldStaker, err := newBOLDStaker(
			ctx,
			m.oldStaker.config,
			m.oldStaker.rollupAddress,
			*m.oldStaker.getCallOpts(ctx),
			auth,
			m.oldStaker.client,
			m.oldStaker.blockValidator,
			m.oldStaker.statelessBlockValidator,
			&m.oldStaker.config.BOLD,
			m.oldStaker.wallet.DataPoster(),
			m.oldStaker.wallet,
		)
		if err != nil {
			return err
		}
		m.boldStaker = boldStaker
		return m.boldStaker.Initialize(ctx)
	}
	return m.oldStaker.Initialize(ctx)
}

func (m *MultiProtocolStaker) Start(ctxIn context.Context) {
	m.StopWaiter.Start(ctxIn, m)
	if m.oldStaker.Strategy() != WatchtowerStrategy {
		m.oldStaker.wallet.Start(ctxIn)
	}
	if m.boldStaker != nil {
		m.boldStaker.Start(ctxIn)
	} else {
		m.oldStaker.Start(ctxIn)
	}
	stakerSwitchInterval := time.Second * 12
	m.CallIteratively(func(ctx context.Context) time.Duration {
		switchedToBoldProtocol, err := m.checkAndSwitchToBoldStaker(ctxIn)
		if err != nil {
			log.Error("staker: error in checking switch to bold staker", "err", err)
			return stakerSwitchInterval
		}
		if switchedToBoldProtocol {
			// Ready to stop the old staker.
			m.oldStaker.StopOnly()
			m.StopOnly()
		}
		return stakerSwitchInterval
	})
}

func (m *MultiProtocolStaker) isBoldActive(ctx context.Context) (bool, common.Address, error) {
	var addr common.Address
	if !m.oldStaker.config.BOLD.Enable {
		return false, addr, nil
	}
	callOpts := m.oldStaker.getCallOpts(ctx)
	rollupAddress, err := m.bridge.Rollup(callOpts)
	if err != nil {
		return false, addr, err
	}
	userLogic, err := oldrollupgen.NewRollupUserLogic(rollupAddress, m.oldStaker.client)
	if err != nil {
		return false, addr, err
	}
	_, err = userLogic.ExtraChallengeTimeBlocks(callOpts)
	// ExtraChallengeTimeBlocks does not exist in the the bold protocol.
	return err != nil, rollupAddress, nil
}

func (m *MultiProtocolStaker) checkAndSwitchToBoldStaker(ctx context.Context) (bool, error) {
	shouldSwitch, rollupAddress, err := m.isBoldActive(ctx)
	if err != nil {
		return false, err
	}
	if !shouldSwitch {
		return false, nil
	}
	txBuilder, err := txbuilder.NewBuilder(m.oldStaker.wallet)
	if err != nil {
		return false, err
	}
	auth, err := txBuilder.Auth(ctx)
	if err != nil {
		return false, err
	}
	boldStaker, err := newBOLDStaker(
		ctx,
		m.oldStaker.config,
		rollupAddress,
		*m.oldStaker.getCallOpts(ctx),
		auth,
		m.oldStaker.client,
		m.oldStaker.blockValidator,
		m.oldStaker.statelessBlockValidator,
		&m.oldStaker.config.BOLD,
		m.oldStaker.wallet.DataPoster(),
		m.oldStaker.wallet,
	)
	if err != nil {
		return false, err
	}
	boldStaker.Start(ctx)
	return true, nil
}
