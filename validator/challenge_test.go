//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package validator

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"math/big"
	"path"
	"runtime"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/offchainlabs/arbstate/solgen/go/challengegen"
	"github.com/offchainlabs/arbstate/solgen/go/ospgen"
)

func DeployOneStepProofEntry(t *testing.T, auth *bind.TransactOpts, client bind.ContractBackend) common.Address {
	osp0, _, _, err := ospgen.DeployOneStepProver0(auth, client)
	if err != nil {
		t.Fatal(err)
	}
	ospMem, _, _, err := ospgen.DeployOneStepProverMemory(auth, client)
	if err != nil {
		t.Fatal(err)
	}
	ospMath, _, _, err := ospgen.DeployOneStepProverMath(auth, client)
	if err != nil {
		t.Fatal(err)
	}
	ospHostIo, _, _, err := ospgen.DeployOneStepProverHostIo(auth, client, common.Address{}, common.Address{})
	if err != nil {
		t.Fatal(err)
	}
	ospEntry, _, _, err := ospgen.DeployOneStepProofEntry(auth, client, osp0, ospMem, ospMath, ospHostIo)
	if err != nil {
		t.Fatal(err)
	}
	return ospEntry
}

func CreateChallenge(
	t *testing.T,
	auth *bind.TransactOpts,
	client bind.ContractBackend,
	ospEntry common.Address,
	startMachineHash common.Hash,
	endMachineHash common.Hash,
	asserter common.Address,
	challenger common.Address,
) (*challengegen.MockResultReceiver, common.Address) {
	resultReceiverAddr, _, resultReceiver, err := challengegen.DeployMockResultReceiver(auth, client)
	if err != nil {
		t.Fatal(err)
	}

	var startHashBytes [32]byte
	var endHashBytes [32]byte
	copy(startHashBytes[:], startMachineHash[:])
	copy(endHashBytes[:], endMachineHash[:])
	challenge, _, _, err := challengegen.DeploySingleExecutionChallenge(
		auth,
		client,
		ospEntry,
		resultReceiverAddr,
		challengegen.ExecutionContext{
			MaxInboxMessagesRead: new(big.Int).SetUint64(^uint64(0)),
		},
		[2][32]byte{startHashBytes, endHashBytes},
		asserter,
		challenger,
		big.NewInt(100),
		big.NewInt(100),
	)
	if err != nil {
		t.Fatal(err)
	}

	return resultReceiver, challenge
}

func createTransactOpts(t *testing.T) *bind.TransactOpts {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	opts, err := bind.NewKeyedTransactorWithChainID(key, big.NewInt(1337))
	if err != nil {
		t.Fatal(err)
	}
	return opts
}

func createGenesisAlloc(accts []*bind.TransactOpts) core.GenesisAlloc {
	alloc := make(core.GenesisAlloc)
	for _, opts := range accts {
		alloc[opts.From] = core.GenesisAccount{
			Balance: big.NewInt(10_000_000),
		}
	}
	return alloc
}

func runChallengeTest(t *testing.T, wasmPath string, asserterIsCorrect bool) {
	deployer := createTransactOpts(t)
	asserter := createTransactOpts(t)
	challenger := createTransactOpts(t)
	alloc := createGenesisAlloc([]*bind.TransactOpts{deployer, asserter, challenger})
	backend := backends.NewSimulatedBackend(alloc, 15_000_000)
	ospEntry := DeployOneStepProofEntry(t, deployer, backend)
	ctx := context.Background()

	machine, err := LoadSimpleMachine(wasmPath)
	if err != nil {
		t.Fatal(err)
	}

	endMachine := machine.Clone()
	err = endMachine.Step(ctx, ^uint64(0))
	if err != nil {
		t.Fatal(err)
	}

	startMachineHash := machine.Hash()
	endMachineHash := endMachine.Hash()

	resultReceiver, challenge := CreateChallenge(
		t,
		deployer,
		backend,
		ospEntry,
		startMachineHash,
		endMachineHash,
		asserter.From,
		challenger.From,
	)

	backend.Commit()

	var asserterMachine MachineInterface = NewIncorrectMachine(machine.Clone(), 20)
	var challengerMachine MachineInterface = machine.Clone()
	expectedWinner := challenger.From
	if asserterIsCorrect {
		asserterMachine, challengerMachine = challengerMachine, asserterMachine
		expectedWinner = asserter.From
	}

	asserterManager, err := NewExecutionChallengeManager(ctx, backend, asserter, challenge, 0, asserterMachine, 4)
	if err != nil {
		t.Fatal(err)
	}
	challengerManager, err := NewExecutionChallengeManager(ctx, backend, challenger, challenge, 0, challengerMachine, 4)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 100; i++ {
		if i%2 == 0 {
			err = challengerManager.Act(ctx)
			if err != nil {
				t.Fatal(err)
			}
		} else {
			err = asserterManager.Act(ctx)
			if err != nil {
				t.Fatal(err)
			}
		}
	}

	winner, err := resultReceiver.Winner(&bind.CallOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if winner == (common.Address{}) {
		t.Fatal("challenge timed out without winner")
	}
	if winner != expectedWinner {
		t.Fatal("wrong party won challenge")
	}
}

func TestChallengeToOSP(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	runChallengeTest(t, path.Join(path.Dir(filename), "../arbitrator/prover/test-cases/global-state.wasm"), false)
}
