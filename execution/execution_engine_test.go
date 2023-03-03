package execution

import "testing"

func TestExecutionEngine(t *testing.T) {
	maxInstructions := uint64(71)
	blockGen := NewBlockGenerator(maxInstructions)
	bh0 := blockGen.BlockHash(0)
	bh1 := blockGen.BlockHash(1)
	bh99 := blockGen.BlockHash(99)
	if bh0 == bh1 || bh0 == bh99 || bh1 == bh99 {
		t.Fatal()
	}

	engine99, err := blockGen.NewExecutionEngine(99, DefaultEngineConfig())
	if err != nil {
		t.Fatal(err)
	}
	numSteps := engine99.NumSteps()
	if numSteps == 0 || numSteps > maxInstructions {
		t.Fatal(numSteps)
	}

	for i := uint64(0); i < numSteps; i++ {
		thisState, err := engine99.StateAfter(i)
		if err != nil {
			t.Fatal(err, i)
		}
		nextState, err := thisState.NextState()
		if err != nil {
			t.Fatal(err, i)
		}
		nextDirect, err := engine99.StateAfter(i + 1)
		if err != nil {
			t.Fatal(err, i)
		}
		if nextState.Hash() != nextDirect.Hash() {
			t.Fatal(i)
		}
		osp, err := thisState.OneStepProof()
		if err != nil {
			t.Fatal(err, i)
		}
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
	if _, err := engine99.StateAfter(engine99.NumSteps() + 1); err == nil {
		t.Fatal()
	}
	if _, err := blockGen.NewExecutionEngine(0, DefaultEngineConfig()); err == nil {
		t.Fatal()
	}
}
