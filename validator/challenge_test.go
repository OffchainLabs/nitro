//
// Copyright 2021-2022, Offchain Labs, Inc. All rights reserved.
//

package validator

import (
	"context"
	"math/big"
	"os"
	"path"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/solgen/go/mocksgen"
	"github.com/offchainlabs/nitro/solgen/go/ospgen"
)

func DeployOneStepProofEntry(t *testing.T, auth *bind.TransactOpts, client bind.ContractBackend) common.Address {
	osp0, _, _, err := ospgen.DeployOneStepProver0(auth, client)
	Require(t, err)

	ospMem, _, _, err := ospgen.DeployOneStepProverMemory(auth, client)
	Require(t, err)

	ospMath, _, _, err := ospgen.DeployOneStepProverMath(auth, client)
	Require(t, err)

	ospHostIo, _, _, err := ospgen.DeployOneStepProverHostIo(auth, client)
	Require(t, err)

	ospEntry, _, _, err := ospgen.DeployOneStepProofEntry(auth, client, osp0, ospMem, ospMath, ospHostIo)
	Require(t, err)
	return ospEntry
}

func CreateChallenge(
	t *testing.T,
	auth *bind.TransactOpts,
	client bind.ContractBackend,
	ospEntry common.Address,
	startMachineHash common.Hash,
	endMachineHash common.Hash,
	endMachineSteps uint64,
	asserter common.Address,
	challenger common.Address,
) (*mocksgen.MockResultReceiver, common.Address) {
	resultReceiverAddr, _, resultReceiver, err := mocksgen.DeployMockResultReceiver(auth, client, common.Address{})
	Require(t, err)

	var startHashBytes [32]byte
	var endHashBytes [32]byte
	copy(startHashBytes[:], startMachineHash[:])
	copy(endHashBytes[:], endMachineHash[:])
	challenge, _, _, err := mocksgen.DeploySingleExecutionChallenge(
		auth,
		client,
		ospEntry,
		resultReceiverAddr,
		^uint64(0),
		[2][32]byte{startHashBytes, endHashBytes},
		big.NewInt(int64(endMachineSteps)),
		asserter,
		challenger,
		big.NewInt(100),
		big.NewInt(100),
	)
	Require(t, err)

	return resultReceiver, challenge
}

func createTransactOpts(t *testing.T) *bind.TransactOpts {
	key, err := crypto.GenerateKey()
	Require(t, err)

	opts, err := bind.NewKeyedTransactorWithChainID(key, big.NewInt(1337))
	Require(t, err)
	return opts
}

func createGenesisAlloc(accts ...*bind.TransactOpts) core.GenesisAlloc {
	alloc := make(core.GenesisAlloc)
	amount := big.NewInt(10)
	amount.Exp(amount, big.NewInt(20), nil)
	for _, opts := range accts {
		alloc[opts.From] = core.GenesisAccount{
			Balance: new(big.Int).Set(amount),
		}
	}
	return alloc
}

func runChallengeTest(
	t *testing.T,
	wasmPath string,
	wasmLibPaths []string,
	steps uint64,
	asserterIsCorrect bool,
	timeBetweenSteps time.Duration,
	shouldTimeout bool,
) {
	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(false)))
	glogger.Verbosity(log.LvlDebug)
	log.Root().SetHandler(glogger)

	ctx := context.Background()
	deployer := createTransactOpts(t)
	asserter := createTransactOpts(t)
	challenger := createTransactOpts(t)
	alloc := createGenesisAlloc(deployer, asserter, challenger)
	backend := backends.NewSimulatedBackend(alloc, 1_000_000_000)
	backend.Commit()

	ospEntry := DeployOneStepProofEntry(t, deployer, backend)
	backend.Commit()

	machine, err := LoadSimpleMachine(wasmPath, wasmLibPaths)
	Require(t, err)

	endMachine := machine.Clone()
	Require(t, endMachine.Step(ctx, ^uint64(0)))

	startMachineHash := machine.Hash()
	endMachineHash := endMachine.Hash()
	endMachineSteps := endMachine.GetStepCount()
	if !asserterIsCorrect {
		endMachineHash = IncorrectMachineHash(endMachineHash)
	}

	resultReceiver, challengeManager := CreateChallenge(
		t,
		deployer,
		backend,
		ospEntry,
		startMachineHash,
		endMachineHash,
		endMachineSteps,
		asserter.From,
		challenger.From,
	)

	backend.Commit()

	var asserterMachine MachineInterface = NewIncorrectMachine(machine.Clone(), steps)
	var challengerMachine MachineInterface = machine.Clone()
	expectedWinner := challenger.From
	if asserterIsCorrect {
		asserterMachine, challengerMachine = challengerMachine, asserterMachine
		expectedWinner = asserter.From
	}

	asserterManager, err := NewExecutionChallengeManager(
		backend,
		asserter,
		challengeManager,
		1,
		asserterMachine,
		0,
		4,
		12,
	)
	Require(t, err)

	challengerManager, err := NewExecutionChallengeManager(
		backend,
		challenger,
		challengeManager,
		1,
		challengerMachine,
		0,
		4,
		12,
	)
	Require(t, err)

	for i := 0; i < 100; i++ {
		err = backend.AdjustTime(timeBetweenSteps)
		backend.Commit()

		var currentCorrect bool
		if i%2 == 0 {
			_, err = challengerManager.Act(ctx)
			currentCorrect = !asserterIsCorrect
		} else {
			_, err = asserterManager.Act(ctx)
			currentCorrect = asserterIsCorrect
		}
		if err != nil {
			if shouldTimeout && strings.Contains(err.Error(), "CHAL_DEADLINE") {
				t.Log("challenge completed in timeout")
				return
			}
			if !currentCorrect &&
				(strings.Contains(err.Error(), "lost challenge") || strings.Contains(err.Error(), "SAME_OSP_END")) {
				if shouldTimeout {
					t.Fatal("expected challenge to end in timeout")
				}
				t.Log("challenge completed! asserter hit expected error:", err)
				return
			}
			t.Fatal(err)
		}

		backend.Commit()

		winner, err := resultReceiver.Winner(&bind.CallOpts{})
		Require(t, err)

		if winner == (common.Address{}) {
			continue
		}
		if winner != expectedWinner {
			t.Fatal("wrong party won challenge")
		}
	}

	t.Fatal("challenge timed out without winner")
}

var wasmDir = (func() string {
	_, filename, _, _ := runtime.Caller(0)
	return path.Join(path.Dir(filename), "../arbitrator/prover/test-cases/")
})()

func TestChallengeToOSP(t *testing.T) {
	runChallengeTest(t, path.Join(wasmDir, "global-state.wasm"), []string{path.Join(wasmDir, "global-state-wrapper.wasm")}, 500, false, 0, false)
}

func TestChallengeToFailedOSP(t *testing.T) {
	runChallengeTest(t, path.Join(wasmDir, "global-state.wasm"), []string{path.Join(wasmDir, "global-state-wrapper.wasm")}, 500, true, 0, false)
}

func TestChallengeToErroredOSP(t *testing.T) {
	runChallengeTest(t, path.Join(wasmDir, "const.wasm"), nil, 23, false, 0, false)
}

func TestChallengeToFailedErroredOSP(t *testing.T) {
	runChallengeTest(t, path.Join(wasmDir, "const.wasm"), nil, 23, true, 0, false)
}

func TestChallengeToTimeout(t *testing.T) {
	runChallengeTest(t, path.Join(wasmDir, "global-state.wasm"), []string{path.Join(wasmDir, "global-state-wrapper.wasm")}, 500, false, time.Second*40, true)
}
