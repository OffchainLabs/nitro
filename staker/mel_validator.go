package staker

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
	melrecorder "github.com/offchainlabs/nitro/arbnode/mel/recorder"
	melrunner "github.com/offchainlabs/nitro/arbnode/mel/runner"
	"github.com/offchainlabs/nitro/util/stopwaiter"
	"github.com/offchainlabs/nitro/validator"
)

type MELValidatorConfig struct {
	FailureIsFatal bool
	ValidationPoll time.Duration
}

type MELValidatorConfigFetcher func() *MELValidatorConfig

type MELValidator struct {
	stopwaiter.StopWaiter
	sync.RWMutex

	config           MELValidatorConfigFetcher
	messageExtractor *melrunner.MessageExtractor
	recorder         *melrecorder.MELRecorder

	lastValidGS   validator.GoGlobalState
	chainCaughtUp bool

	entryCreatorChan  chan struct{}
	entryRecorderChan chan struct{}
	entrySenderChan   chan struct{}
	entryCleanupChan  chan struct{}
	fatalErr          chan<- error
}

func NewMELValidator(
	ctx context.Context,
	configFetcher MELValidatorConfigFetcher,
	messageExtractor *melrunner.MessageExtractor,
	recorder *melrecorder.MELRecorder,
	fatalErr chan<- error,
) *MELValidator {
	return &MELValidator{
		config:            configFetcher,
		messageExtractor:  messageExtractor,
		recorder:          recorder,
		entryCreatorChan:  make(chan struct{}, 1),
		entryRecorderChan: make(chan struct{}, 1),
		entrySenderChan:   make(chan struct{}, 1),
		entryCleanupChan:  make(chan struct{}, 1),
		fatalErr:          fatalErr,
	}
}

func (v *MELValidator) Start(ctxIn context.Context) error {
	v.StopWaiter.Start(ctxIn, v)
	v.LaunchThread(v.LaunchWhenCaughtUp)
	// TODO: Print out validation progress.
	return nil
}

func (v *MELValidator) StopAndWait() {
	v.StopWaiter.StopAndWait()
}

func (v *MELValidator) LaunchWhenCaughtUp(ctx context.Context) {
	for {
		caughtUp, err := v.isCaughtUp(ctx)
		if err != nil {
			log.Error("validator got error waiting for chain to catch up. Consider restarting with dangerous.reset-block-validation", "err", err)
		}
		if caughtUp {
			break
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(v.config().ValidationPoll):
		}
	}
	err := stopwaiter.CallIterativelyWith(&v.StopWaiterSafe, v.iterativeValidationEntryCreator, v.entryCreatorChan)
	if err != nil {
		v.possiblyFatal(err)
	}
	err = stopwaiter.CallIterativelyWith(&v.StopWaiterSafe, v.iterativeValidationEntryRecorder, v.entryRecorderChan)
	if err != nil {
		v.possiblyFatal(err)
	}
	err = stopwaiter.CallIterativelyWith(&v.StopWaiterSafe, v.iterativeValidationEntrySender, v.entrySenderChan)
	if err != nil {
		v.possiblyFatal(err)
	}
	err = stopwaiter.CallIterativelyWith(&v.StopWaiterSafe, v.iterativeValidationEntryCleanup, v.entryCleanupChan)
	if err != nil {
		v.possiblyFatal(err)
	}
}

func (v *MELValidator) isCaughtUp(ctx context.Context) (bool, error) {
	v.Lock()
	defer v.Unlock()
	if v.chainCaughtUp {
		return true, nil
	}
	if v.lastValidGS.Batch == 0 {
		return false, errors.New("lastValid not initialized. cannot validate genesis")
	}
	melMsgCount, err := v.messageExtractor.GetMsgCount(ctx)
	if err != nil {
		return false, err
	}
	requiredMsgCount := v.lastValidGS.PosInBatch
	if melMsgCount < requiredMsgCount {
		return false, nil
	}
	// TODO: uncomment after MELStateRoot is added to GS
	// melState, err := v.messageExtractor.GetStateByParentChainBlockHash(ctx, v.lastValidGS.BlockHash)
	// if err != nil {
	// 	return false, err
	// }
	// melStateRoot := melState.Hash()
	// if melStateRoot != v.lastValidGS.MELStateRoot {
	// 	return false, fmt.Errorf("melstate root: %v doesnt match the one in db: %v", v.lastValidGS.MELStateRoot, melStateRoot)
	// }
	return true, nil
}

func (v *MELValidator) iterativeValidationEntryCreator(ctx context.Context, _ struct{}) time.Duration {
	return 0
}

func (v *MELValidator) iterativeValidationEntryRecorder(ctx context.Context, _ struct{}) time.Duration {
	return 0
}

func (v *MELValidator) iterativeValidationEntrySender(ctx context.Context, _ struct{}) time.Duration {
	return 0
}

func (v *MELValidator) iterativeValidationEntryCleanup(ctx context.Context, _ struct{}) time.Duration {
	return 0
}

func (v *MELValidator) possiblyFatal(err error) {
	if v.Stopped() {
		return
	}
	if err == nil {
		return
	}
	log.Error("Error during validation", "err", err)
	if v.config().FailureIsFatal {
		select {
		case v.fatalErr <- err:
		default:
		}
	}
}
