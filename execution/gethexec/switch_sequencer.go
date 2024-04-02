package gethexec

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/arbitrum_types"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

const (
	SequencingMode_Espresso    = 0
	SequencingMode_Centralized = 1
)

type SwitchSequencer struct {
	stopwaiter.StopWaiter

	centralized *Sequencer
	espresso    *EspressoSequencer

	mode                     int
	maxHotshotDirftTime      time.Duration
	consecutiveHotshotBlocks int
	hotshotTimeFrame         time.Duration

	lastSeenHotShotBlock uint64
}

func NewSequencerSwitch(centralized *Sequencer, espresso *EspressoSequencer, configFetcher SequencerConfigFetcher) (*SwitchSequencer, error) {
	config := configFetcher()
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &SwitchSequencer{
		centralized:              centralized,
		espresso:                 espresso,
		mode:                     SequencingMode_Espresso,
		maxHotshotDirftTime:      config.MaxHotshotDriftTime,
		consecutiveHotshotBlocks: config.ConsecutiveHotshotBlocks,
		hotshotTimeFrame:         config.HotshotTimeFrame,
		lastSeenHotShotBlock:     0,
	}, nil
}

func (s *SwitchSequencer) IsRunningEspressoMode() bool {
	return s.mode == SequencingMode_Espresso
}

func (s *SwitchSequencer) SwitchToEspresso(ctx context.Context) error {
	if s.mode == SequencingMode_Espresso {
		return nil
	}
	s.mode = SequencingMode_Espresso
	s.centralized.StopAndWait()
	return s.espresso.Start(ctx)
}

func (s *SwitchSequencer) SwitchToCentralized(ctx context.Context) error {
	if s.mode == SequencingMode_Centralized {
		return nil
	}
	s.mode = SequencingMode_Centralized
	s.espresso.StopAndWait()
	return s.espresso.Start(ctx)
}

func (s *SwitchSequencer) getRunningSequencer() TransactionPublisher {
	if s.IsRunningEspressoMode() {
		return s.espresso
	}
	return s.centralized
}

func (s *SwitchSequencer) PublishTransaction(ctx context.Context, tx *types.Transaction, options *arbitrum_types.ConditionalOptions) error {
	return s.getRunningSequencer().PublishTransaction(ctx, tx, options)
}

func (s *SwitchSequencer) CheckHealth(ctx context.Context) error {
	return s.getRunningSequencer().CheckHealth(ctx)
}

func (s *SwitchSequencer) Initialize(ctx context.Context) error {
	return s.getRunningSequencer().Initialize(ctx)
}

func (s *SwitchSequencer) Start(ctx context.Context) error {
	err := s.getRunningSequencer().Start(ctx)
	if err != nil {
		return err
	}
	s.CallIteratively(func(ctx context.Context) time.Duration {
		now := time.Now()
		if s.IsRunningEspressoMode() {
			if s.espresso.lastCreated.Add(s.maxHotshotDirftTime).Before(now) {
				_ = s.SwitchToCentralized(ctx)
			}
			return s.hotshotTimeFrame
		}

		b, err := s.espresso.hotShotState.client.FetchLatestBlockHeight(ctx)
		if err != nil {
			return 100 * time.Second
		}
		if s.lastSeenHotShotBlock == 0 {
			s.lastSeenHotShotBlock = b
			return s.hotshotTimeFrame
		}
		if b-s.lastSeenHotShotBlock <= uint64(s.consecutiveHotshotBlocks) {
			s.lastSeenHotShotBlock = 0
			_ = s.SwitchToEspresso(ctx)
			return 0
		}
		s.lastSeenHotShotBlock = b
		return s.hotshotTimeFrame
	})

	return nil
}

func (s *SwitchSequencer) StopAndWait() {
	s.getRunningSequencer().StopAndWait()
	s.StopWaiter.StopAndWait()
}

func (s *SwitchSequencer) Started() bool {
	return s.getRunningSequencer().Started()
}
