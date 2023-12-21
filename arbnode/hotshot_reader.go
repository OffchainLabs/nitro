package arbnode

import (
	"fmt"
	"math/big"

	"github.com/EspressoSystems/espresso-sequencer-go/hotshot"
	espressoTypes "github.com/EspressoSystems/espresso-sequencer-go/types"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

type HotShotReader struct {
	HotShot hotshot.Hotshot
}

func NewHotShotReader(hotShotAddr common.Address, l1client bind.ContractBackend) (*HotShotReader, error) {
	hotshot, err := hotshot.NewHotshot(hotShotAddr, l1client)
	if err != nil {
		return nil, err
	}

	return &HotShotReader{
		HotShot: *hotshot,
	}, nil
}

// L1HotShotCommitmentFromHeight returns a HotShot commitments to a sequencer block
// This is used in the derivation pipeline to validate sequencer batches in Espresso mode
func (h *HotShotReader) L1HotShotCommitmentFromHeight(blockHeight uint64) (*espressoTypes.Commitment, error) {
	var comm espressoTypes.Commitment
	// Check if the requested commitments are even available yet on L1.
	contractBlockHeight, err := h.HotShot.HotshotCaller.BlockHeight(nil)
	if err != nil {
		return nil, err
	}
	if contractBlockHeight.Cmp(big.NewInt(int64(blockHeight))) < 0 {
		return nil, fmt.Errorf("commitment at block height %d is unavailable (current contract block height %d)", blockHeight, contractBlockHeight)
	}

	commAsInt, err := h.HotShot.HotshotCaller.Commitments(nil, big.NewInt(int64(blockHeight)))
	if err != nil {
		return nil, err
	}
	if commAsInt.Cmp(big.NewInt(0)) == 0 {
		// A commitment of 0 indicates that this commitment hasn't been set yet in the contract.
		// Since we checked the contract block height above, this can only happen if there was
		// a reorg on L1 just now. In this case, return an error rather than reporting
		// definitive commitments. The caller will retry and we will succeed eventually when we
		// manage to get a consistent snapshot of the L1.
		//
		// Note that in all other reorg cases, where the L1 reorgs but we read a nonzero
		// commitment, we are fine, since the HotShot contract will only ever record a single
		// ledger, consistent across all L1 forks, determined by HotShot consensus. The only
		// question is whether the recorded ledger extends far enough for the commitments we're
		// trying to read on the current fork of L1.
		return nil, fmt.Errorf("read 0 for commitment %d at block height %d, this indicates an L1 reorg", blockHeight, contractBlockHeight)
	}

	comm, err = espressoTypes.CommitmentFromUint256(espressoTypes.NewU256().SetBigInt(commAsInt))

	if err != nil {
		return nil, err
	}

	log.Info("Sucessfully read commitment", "blockHeight", blockHeight, "commitment", commAsInt.String())

	return &comm, nil
}
