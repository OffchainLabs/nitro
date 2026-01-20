package melrecording

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/arbnode/mel"
	"github.com/offchainlabs/nitro/arbos/merkleAccumulator"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/daprovider"
)

// InitializeRecordingMsgPreimages initializes the given state's msgAcc to record preimages related
// to the extracted messages needed for MEL validation into the given preimages map
func InitializeRecordingMsgPreimages(state *mel.State, preimages daprovider.PreimagesMap) error {
	if preimages == nil {
		return errors.New("preimages recording destination cannot be nil")
	}
	if _, ok := preimages[arbutil.Keccak256PreimageType]; !ok {
		preimages[arbutil.Keccak256PreimageType] = make(map[common.Hash][]byte)
	}
	acc, err := merkleAccumulator.NewNonpersistentMerkleAccumulatorFromPartials(mel.ToPtrSlice(state.MessageMerklePartials))
	if err != nil {
		return err
	}
	acc.RecordPreimagesTo(preimages[arbutil.Keccak256PreimageType])
	state.SetMsgsAcc(acc)
	return nil
}
