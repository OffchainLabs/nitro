package statemanager

import (
	"github.com/OffchainLabs/challenge-protocol-v2/execution"
	"testing"
)

func TestSubchallengeCommitments(t *testing.T) {
	maxInstructions := uint64(71)
	blockGen := execution.NewBlockGenerator(maxInstructions)
	bh0 := blockGen.BlockHash(0)
	bh1 := blockGen.BlockHash(1)
	bh99 := blockGen.BlockHash(99)
	if bh0 == bh1 || bh0 == bh99 || bh1 == bh99 {
		t.Fatal()
	}

	engine99, err := blockGen.NewExecutionEngine(99)
	if err != nil {
		t.Fatal(err)
	}
	numSteps := engine99.NumSteps()
	if numSteps == 0 || numSteps > maxInstructions {
		t.Fatal(numSteps)
	}
}
