package arbtest

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/validator"
	"github.com/offchainlabs/nitro/validator/server_arb"
	"github.com/offchainlabs/nitro/validator/server_common"
)

func TestEspressoValidation(t *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	locator, err := server_common.NewMachineLocator("")
	if err != nil {
		Fatal(t, err)
	}
	wasmModuleRoot := locator.LatestWasmModuleRoot()
	if (wasmModuleRoot == common.Hash{}) {
		Fatal(t, "latest machine not found")
	}

	data, err := os.ReadFile("espresso-e2e/validation_input2.json")
	Require(t, err)
	var input validator.ValidationInput
	err = json.Unmarshal(data, &input)
	Require(t, err)

	machine, err := server_arb.CreateTestArbMachine(ctx, locator, &input)
	Require(t, err)
	machine.Step(ctx, 10000000000)
	if machine.IsErrored() {
		panic("")
	}
}
