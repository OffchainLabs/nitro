// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package staker

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
	"github.com/offchainlabs/nitro/validator"
	"github.com/offchainlabs/nitro/validator/server_arb"
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
	ctx context.Context,
	auth *bind.TransactOpts,
	client bind.ContractBackend,
	ospEntry common.Address,
	inputMachine server_arb.MachineInterface,
	maxInboxMessage uint64,
	asserter common.Address,
	challenger common.Address,
) (*mocksgen.MockResultReceiver, common.Address) {
	resultReceiverAddr, _, resultReceiver, err := mocksgen.DeployMockResultReceiver(auth, client, common.Address{})
	Require(t, err)

	machine := inputMachine.CloneMachineInterface()
	startMachineHash := machine.Hash()

	Require(t, machine.Step(ctx, ^uint64(0)))

	endMachineHash := machine.Hash()
	endMachineSteps := machine.GetStepCount()

	var startHashBytes [32]byte
	var endHashBytes [32]byte
	copy(startHashBytes[:], startMachineHash[:])
	copy(endHashBytes[:], endMachineHash[:])
	challenge, _, _, err := mocksgen.DeploySingleExecutionChallenge(
		auth,
		client,
		ospEntry,
		resultReceiverAddr,
		maxInboxMessage,
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
	baseMachine *server_arb.ArbitratorMachine,
	incorrectMachine server_arb.MachineInterface,
	asserterIsCorrect bool,
	testTimeout bool,
	maxInboxMessage uint64,
) {
	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(false)))
	glogger.Verbosity(log.LvlDebug)
	log.Root().SetHandler(glogger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	deployer := createTransactOpts(t)
	asserter := createTransactOpts(t)
	challenger := createTransactOpts(t)
	alloc := createGenesisAlloc(deployer, asserter, challenger)
	backend := backends.NewSimulatedBackend(alloc, 1_000_000_000)
	backend.Commit()

	ospEntry := DeployOneStepProofEntry(t, deployer, backend)
	backend.Commit()

	var asserterMachine, challengerMachine server_arb.MachineInterface
	var expectedWinner common.Address
	if asserterIsCorrect {
		expectedWinner = asserter.From
		asserterMachine = baseMachine.Clone()
		challengerMachine = incorrectMachine
	} else {
		expectedWinner = challenger.From
		asserterMachine = incorrectMachine
		challengerMachine = baseMachine.Clone()
	}

	resultReceiver, challengeManager := CreateChallenge(
		t,
		ctx,
		deployer,
		backend,
		ospEntry,
		asserterMachine,
		maxInboxMessage,
		asserter.From,
		challenger.From,
	)

	backend.Commit()

	asserterRun, err := server_arb.NewExecutionRun(ctx,
		func(context.Context) (server_arb.MachineInterface, error) { return asserterMachine, nil },
		&server_arb.DefaultMachineCacheConfig)
	Require(t, err)

	asserterManager, err := NewExecutionChallengeManager(
		backend,
		asserter,
		challengeManager,
		1,
		asserterRun,
		0,
		12,
	)
	Require(t, err)

	challengerRun, err := server_arb.NewExecutionRun(ctx,
		func(context.Context) (server_arb.MachineInterface, error) { return challengerMachine, nil },
		&server_arb.DefaultMachineCacheConfig)
	Require(t, err)
	challengerManager, err := NewExecutionChallengeManager(
		backend,
		challenger,
		challengeManager,
		1,
		challengerRun,
		0,
		12,
	)
	Require(t, err)

	for i := 0; i < 100; i++ {
		if testTimeout {
			err = backend.AdjustTime(time.Second * 40)
		}
		Require(t, err)
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
			if testTimeout && strings.Contains(err.Error(), "CHAL_DEADLINE") {
				t.Log("challenge completed in timeout")
				return
			}
			if !currentCorrect &&
				(strings.Contains(err.Error(), "lost challenge") || strings.Contains(err.Error(), "SAME_OSP_END")) {
				if testTimeout {
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

func createBaseMachine(t *testing.T, wasmname string, wasmModules []string) *server_arb.ArbitratorMachine {
	_, filename, _, _ := runtime.Caller(0)
	wasmDir := path.Join(path.Dir(filename), "../arbitrator/prover/test-cases/")

	wasmPath := path.Join(wasmDir, wasmname)

	var modulePaths []string
	for _, moduleName := range wasmModules {
		modulePaths = append(modulePaths, path.Join(wasmDir, moduleName))
	}

	machine, err := server_arb.LoadSimpleMachine(wasmPath, modulePaths)
	Require(t, err)

	return machine
}

func TestChallengeToOSP(t *testing.T) {
	machine := createBaseMachine(t, "global-state.wasm", []string{"global-state-wrapper.wasm"})
	IncorrectMachine := server_arb.NewIncorrectMachine(machine, 200)
	runChallengeTest(t, machine, IncorrectMachine, false, false, 0)
}

func TestChallengeToFailedOSP(t *testing.T) {
	machine := createBaseMachine(t, "global-state.wasm", []string{"global-state-wrapper.wasm"})
	IncorrectMachine := server_arb.NewIncorrectMachine(machine, 200)
	runChallengeTest(t, machine, IncorrectMachine, true, false, 0)
}

func TestChallengeToErroredOSP(t *testing.T) {
	machine := createBaseMachine(t, "const.wasm", nil)
	IncorrectMachine := server_arb.NewIncorrectMachine(machine, 10)
	runChallengeTest(t, machine, IncorrectMachine, false, false, 0)
}

func TestChallengeToFailedErroredOSP(t *testing.T) {
	machine := createBaseMachine(t, "const.wasm", nil)
	IncorrectMachine := server_arb.NewIncorrectMachine(machine, 10)
	runChallengeTest(t, machine, IncorrectMachine, true, false, 0)
}

func TestChallengeToTimeout(t *testing.T) {
	machine := createBaseMachine(t, "global-state.wasm", []string{"global-state-wrapper.wasm"})
	IncorrectMachine := server_arb.NewIncorrectMachine(machine, 200)
	runChallengeTest(t, machine, IncorrectMachine, false, true, 0)
}

func TestChallengeToTooFar(t *testing.T) {
	machine := createBaseMachine(t, "read-inboxmsg-10.wasm", []string{"global-state-wrapper.wasm"})
	Require(t, machine.SetGlobalState(validator.GoGlobalState{PosInBatch: 10}))
	incorrectMachine := machine.Clone()
	Require(t, incorrectMachine.AddSequencerInboxMessage(10, []byte{0, 1, 2, 3}))
	runChallengeTest(t, machine, incorrectMachine, false, false, 9)
}

func TestChallengeToFailedTooFar(t *testing.T) {
	machine := createBaseMachine(t, "read-inboxmsg-10.wasm", []string{"global-state-wrapper.wasm"})
	Require(t, machine.SetGlobalState(validator.GoGlobalState{PosInBatch: 10}))
	incorrectMachine := machine.Clone()
	Require(t, machine.AddSequencerInboxMessage(10, []byte{0, 1, 2, 3}))
	runChallengeTest(t, machine, incorrectMachine, true, false, 11)
}
