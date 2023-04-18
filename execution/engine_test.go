package execution

import (
	"testing"

	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

var (
	_ = EngineAtBlock(&Engine{})
	_ = IntermediateStateIterator(&ExecutionState{})
)

func TestWorstCaseBigStepBisections(t *testing.T) {
	hashes := make([]common.Hash, 0)
	hashes = append(hashes, common.Hash{})
	for len(hashes) <= 100 {
		hashes = append(
			hashes,
			crypto.Keccak256Hash(hashes[len(hashes)-1].Bytes()),
		)
	}
	engine99, err := NewExecutionEngine(DefaultMachineConfig(), hashes[98], hashes[100])
	require.NoError(t, err)
	numSteps := engine99.NumOpcodes()
	numBigSteps := engine99.NumBigSteps()
	t.Logf("Number of total steps %d", numSteps)
	t.Logf("Number of big steps: %d", numBigSteps)

	osfHeight := numBigSteps
	var totalBisections int
	for osfHeight != 2 {
		bisectTo, err := util.BisectionPoint(1, osfHeight)
		require.NoError(t, err)
		osfHeight = bisectTo
		totalBisections++
	}
	t.Logf("Total bisections: %d", totalBisections)
	require.Equal(t, 22, totalBisections)
}

func TestExecutionEngine(t *testing.T) {
	t.Skip("A few assumptions about intermediate leaves changed, needs refactoring")
	hashes := make([]common.Hash, 0)
	hashes = append(hashes, common.Hash{})
	for len(hashes) <= 100 {
		hashes = append(
			hashes,
			crypto.Keccak256Hash(hashes[len(hashes)-1].Bytes()),
		)
	}
	engine99, err := NewExecutionEngine(DefaultMachineConfig(), hashes[98], hashes[100])
	if err != nil {
		t.Fatal(err)
	}
	numSteps := engine99.NumOpcodes()
	numHashes := uint64(len(hashes[98:100]))
	if numSteps == 0 || numSteps > numHashes*DefaultMachineConfig().MaxInstructionsPerBlock {
		t.Fatal(numSteps)
	}

	for i := uint64(0); i < 10; i++ {
		thisState, err := engine99.StateAfterSmallSteps(i)
		require.NoError(t, err)
		nextState, err := thisState.NextMachineState()
		require.NoError(t, err)
		nextDirect, err := engine99.StateAfterSmallSteps(i + 1)
		require.NoError(t, err)

		if nextState.Hash() != nextDirect.Hash() {
			t.Fatal(i)
		}
		osp, err := OneStepProof(thisState)
		require.NoError(t, err)

		if !VerifyOneStepProof(thisState.Hash(), nextState.Hash(), osp) {
			t.Fatal(i)
		}

		// verify that bad proofs get rejected
		fakeProof := append([]byte{}, osp...)
		fakeProof = append(fakeProof, 0)
		if VerifyOneStepProof(thisState.Hash(), nextState.Hash(), fakeProof) {
			t.Fatal(i)
		}

		fakeProof = append([]byte{}, osp...)
		fakeProof[19] ^= 1
		if VerifyOneStepProof(thisState.Hash(), nextState.Hash(), fakeProof) {
			t.Fatal(i)
		}

		fakeProof = append([]byte{}, osp...)
		fakeProof[19+32] ^= 1
		if VerifyOneStepProof(thisState.Hash(), nextState.Hash(), fakeProof) {
			t.Fatal(i)
		}

		fakeProof = append([]byte{}, osp...)
		fakeProof[64+3] ^= 1
		if VerifyOneStepProof(thisState.Hash(), nextState.Hash(), fakeProof) {
			t.Fatal(i)
		}

		fakeProof = append([]byte{}, osp...)
		fakeProof = fakeProof[len(fakeProof)-1:]
		if VerifyOneStepProof(thisState.Hash(), nextState.Hash(), fakeProof) {
			t.Fatal(i)
		}
	}
	// check for expected errors
	if _, err := engine99.StateAfterSmallSteps(engine99.NumOpcodes() + 1); err == nil {
		t.Fatal()
	}
}
