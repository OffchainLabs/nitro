package arbtest

import (
	"context"
	"encoding/json"
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/solgen/go/ospgen"
	"github.com/offchainlabs/nitro/solgen/go/test_helpersgen"
	"github.com/offchainlabs/nitro/validator"
	"github.com/offchainlabs/nitro/validator/server_arb"
	"github.com/offchainlabs/nitro/validator/server_common"
)

func TestEspressoOsp(t *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	initialBalance := new(big.Int).Lsh(big.NewInt(1), 200)
	l1Info := NewL1TestInfo(t)
	l1Info.GenerateGenesisAccount("deployer", initialBalance)

	deployerTxOpts := l1Info.GetDefaultTransactOpts("deployer", ctx)

	chainConfig := params.ArbitrumDevTestChainConfig()
	l1Info, l1Backend, _, _ := createTestL1BlockChain(t, l1Info)
	hotshotAddr, tx, hotShotConn, err := test_helpersgen.DeployMockHotShot(&deployerTxOpts, l1Backend)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, l1Backend, tx)
	Require(t, err)

	locator, err := server_common.NewMachineLocator("")
	Require(t, err)
	rollup, _ := DeployOnTestL1(t, ctx, l1Info, l1Backend, chainConfig, locator.LatestWasmModuleRoot(), hotshotAddr)

	ospEntryAddr := common.HexToAddress("0xffd0c2C95214aa9980D7419bd87c260C80Ce2546")

	wasmModuleRoot := locator.LatestWasmModuleRoot()
	if (wasmModuleRoot == common.Hash{}) {
		Fatal(t, "latest machine not found")
	}

	// To generate a valid validation_input, copy these code to
	// the appropriate place in staker/block_validator.go: function advanceValidations
	// and run the espresso_e2e_test.
	/// ```
	// input, _ := validationStatus.Entry.ToInput()
	// file, _ := os.Create("espresso-e2e/validation_input.json")
	// s, _ := json.Marshal(input)
	// file.Write(s)
	// ````
	data, err := os.ReadFile("espresso-e2e/validation_input.json")
	Require(t, err)
	var input validator.ValidationInput
	err = json.Unmarshal(data, &input)
	Require(t, err)

	machine, err := server_arb.CreateTestArbMachine(ctx, locator, &input)
	Require(t, err)
	err = machine.StepUntilReadHotShot(ctx)
	Require(t, err)
	if !machine.IsRunning() {
		t.Fatal("should be still running")
	}
	comm, _ := big.NewInt(0).SetString(common.Hash(input.HotShotCommitment).String(), 0)
	tx, err = hotShotConn.SetCommitment(
		&deployerTxOpts,
		big.NewInt(int64(0)).SetUint64(input.BlockHeight),
		comm,
	)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, l1Backend, tx)
	Require(t, err)

	proof := machine.ProveNextStep()
	beforeHash := machine.Hash()
	err = machine.Step(ctx, uint64(1))
	Require(t, err)
	expectedAfterHash := machine.Hash()

	ospEntry, err := ospgen.NewOneStepProofEntry(ospEntryAddr, l1Backend)
	Require(t, err)
	afterHash, err := ospEntry.ProveOneStep(
		l1Info.GetDefaultCallOpts("deployer", ctx),
		ospgen.ExecutionContext{
			MaxInboxMessagesRead: big.NewInt(1),
			Bridge:               rollup.Bridge,
		},
		// Machine step has no effect on this test.
		// In the contract validation, we didn't use the step
		big.NewInt(10),
		beforeHash,
		proof,
	)
	Require(t, err)

	if afterHash == [32]byte{} {
		t.Fatal("get the empty hash from the L1")
	}

	log.Info("osp entry", "expected hash", expectedAfterHash, "actual", common.Hash(afterHash))
	if expectedAfterHash != afterHash {
		t.Fatal("read hotshot commitment op wrong")
	}

	// load another machine to test the IsHotShotLive Opcode
	machine, err = server_arb.CreateTestArbMachine(ctx, locator, &input)
	Require(t, err)
	err = machine.StepUntilIsHotShotLive(ctx)
	Require(t, err)
	if !machine.IsRunning() {
		t.Fatal("should be still running")
	}

	liveness := input.HotShotLiveness
	tx, err = hotShotConn.SetLiveness(&deployerTxOpts,
		big.NewInt(int64(0)).SetUint64(input.BlockHeight),
		liveness,
	)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, l1Backend, tx)
	Require(t, err)

	livenessProof := machine.ProveNextStep()
	beforeHash = machine.Hash()
	err = machine.Step(ctx, uint64(1))
	Require(t, err)
	expectedAfterHash = machine.Hash()
	afterHash, err = ospEntry.ProveOneStep(
		l1Info.GetDefaultCallOpts("deployer", ctx),
		ospgen.ExecutionContext{
			MaxInboxMessagesRead: big.NewInt(1),
			Bridge:               rollup.Bridge,
		},
		// Machine step has no effect on this test.
		// In the contract validation, we didn't use the step
		big.NewInt(10),
		beforeHash,
		livenessProof,
	)
	Require(t, err)

	if expectedAfterHash != afterHash {
		t.Fatal("isHotShotLive op wrong")
	}
}
