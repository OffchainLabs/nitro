package validator

import (
	"github.com/OffchainLabs/challenge-protocol-v2/execution"
	"github.com/stretchr/testify/require"
	"testing"
)

const (
	maxInstructions = 1 << 43
	bigStepSize     = 1 << 20
)

func TestSubchallengeCommitments(t *testing.T) {
	blockGen := execution.NewBlockGenerator(maxInstructions)
	bh0 := blockGen.BlockHash(0)
	bh1 := blockGen.BlockHash(1)
	bh99 := blockGen.BlockHash(99)
	if bh0 == bh1 || bh0 == bh99 || bh1 == bh99 {
		t.Fatal()
	}
	engine99, err := blockGen.NewExecutionEngine(99, &execution.ExecEngineConfig{
		NumSteps: 1,
	})
	require.NoError(t, err)
	numSteps := engine99.NumSteps()
	var numBigSteps uint64
	if numSteps < bigStepSize {
		numBigSteps = 1
	} else {
		numBigSteps = numSteps / bigStepSize
	}
	t.Log(numSteps, numBigSteps)
}
