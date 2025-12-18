package staker

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/offchainlabs/nitro/arbnode/mel"
	"github.com/offchainlabs/nitro/arbnode/mel/extraction"
	"github.com/offchainlabs/nitro/arbnode/mel/runner"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

type MELValidator struct {
	stopwaiter.StopWaiter
	arbDb            ethdb.KeyValueStore
	l1client         *ethclient.Client
	messageExtractor *melrunner.MessageExtractor
	dapReaders       arbstate.DapReaderSource
}

func NewMELValdidator(messageExtractor *melrunner.MessageExtractor) *MELValidator {
	return &MELValidator{
		messageExtractor: messageExtractor,
	}
}

func (mv *MELValidator) CreateNextValidationEntry(ctx context.Context, startPosition uint64) (*validationEntry, error) {
	if startPosition == 0 {
		return nil, errors.New("trying to create validation entry for zero block number")
	}
	preState, err := mv.messageExtractor.GetState(ctx, startPosition-1)
	if err != nil {
		return nil, err
	}
	delayedMsgRecordingDB := melrunner.NewRecordingDatabase(mv.arbDb)
	recordingDAPReaders := melrunner.NewRecordingDAPReaderSource(ctx, mv.dapReaders)
	for i := startPosition; ; i++ {
		header, err := mv.l1client.HeaderByNumber(ctx, new(big.Int).SetUint64(i))
		if err != nil {
			return nil, err
		}
		// Awaiting recording implementations of logsFetcher and txsFetcher
		state, _, _, _, err := melextraction.ExtractMessages(ctx, preState, header, recordingDAPReaders, delayedMsgRecordingDB, nil, nil)
		if err != nil {
			return nil, fmt.Errorf("error calling melextraction.ExtractMessages in recording mode: %w", err)
		}
		wantState, err := mv.messageExtractor.GetState(ctx, i)
		if err != nil {
			return nil, err
		}
		if state.Hash() != wantState.Hash() {
			return nil, fmt.Errorf("calculated MEL state hash in recording mode doesn't match the one computed in native mode, parentchainBlocknumber: %d", i)
		}
		if mel.WasMessageExtracted(preState, state) {
			break
		}
		preState = state
	}
	preimages := recordingDAPReaders.Preimages()
	delayedPreimages := daprovider.PreimagesMap{
		arbutil.Keccak256PreimageType: delayedMsgRecordingDB.Preimages(),
	}
	daprovider.CopyPreimagesInto(preimages, delayedPreimages)
	return nil, nil
}
