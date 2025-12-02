// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see:
// https://github.com/offchainlabs/nitro/blob/master/LICENSE.md
package stateprovider

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/bold/chain-abstraction"
)

func TestExecutionEngine(t *testing.T) {
	startState := &protocol.ExecutionState{
		GlobalState:   protocol.GoGlobalState{},
		MachineStatus: protocol.MachineStatusFinished,
	}
	machine := NewSimpleMachine(startState, nil)
	for i := uint64(0); i < 100; i++ {
		require.Equal(t, i, machine.CurrentStepNum())

		thisHash := machine.Hash()
		stopped := machine.IsStopped()
		osp, err := machine.OneStepProof()
		require.NoError(t, err)
		err = machine.Step(1)
		require.NoError(t, err)
		nextHash := machine.Hash()

		require.Equal(t, thisHash == nextHash, stopped)

		if !VerifySimpleMachineOneStepProof(thisHash, nextHash, i, nil, osp) {
			t.Fatal(i)
		}

		// verify that bad proofs get rejected
		fakeProof := append([]byte{}, osp...)
		fakeProof = append(fakeProof, 0)
		if VerifySimpleMachineOneStepProof(thisHash, nextHash, i, nil, fakeProof) {
			t.Fatal(i)
		}

		fakeProof = append([]byte{}, osp...)
		fakeProof[0] ^= 1
		if VerifySimpleMachineOneStepProof(thisHash, nextHash, i, nil, fakeProof) {
			t.Fatal(i)
		}

		fakeProof = append([]byte{}, osp...)
		fakeProof = fakeProof[len(fakeProof)-1:]
		if VerifySimpleMachineOneStepProof(thisHash, nextHash, i, nil, fakeProof) {
			t.Fatal(i)
		}

		if thisHash != nextHash && VerifySimpleMachineOneStepProof(thisHash, thisHash, i, nil, osp) {
			t.Fatal(i)
		}
		if VerifySimpleMachineOneStepProof(thisHash, common.Hash{}, i, nil, osp) {
			t.Fatal(i)
		}
	}
}
