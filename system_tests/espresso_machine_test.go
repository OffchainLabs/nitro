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
	"github.com/offchainlabs/nitro/validator/server_jit"
)

func TestEspressoArbMachine(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	locator, err := server_common.NewMachineLocator("")
	if err != nil {
		Fatal(t, err)
	}
	data, err := os.ReadFile("espresso-e2e/validation_input.json")
	Require(t, err)
	var input validator.ValidationInput
	err = json.Unmarshal(data, &input)
	Require(t, err)

	machine, err := server_arb.CreateTestArbMachine(ctx, locator, &input)
	Require(t, err)

	err = machine.StepUntilHostIo(ctx)
	Require(t, err)

	if machine.IsErrored() || !machine.IsRunning() {
		panic("arb machine should be running")
	}

	machine.Step(ctx, 900000000)
	machine.Step(ctx, 900000000)

	if machine.IsErrored() || machine.IsRunning() {
		panic("arb machine should finish all the steps")
	}
}

func TestEspressoJitMachine(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	locator, err := server_common.NewMachineLocator("")
	if err != nil {
		Fatal(t, err)
	}
	data, err := os.ReadFile("espresso-e2e/validation_input.json")
	Require(t, err)
	var input validator.ValidationInput
	err = json.Unmarshal(data, &input)
	Require(t, err)

	config := &server_jit.DefaultJitSpawnerConfig
	config.WasmMemoryUsageLimit = 5 * config.WasmMemoryUsageLimit
	fetcher := func() *server_jit.JitSpawnerConfig { return config }
	spawner, err := server_jit.NewJitSpawner(locator, fetcher, nil)
	Require(t, err)
	_, err = spawner.TestExecute(ctx, &input, common.Hash{})
	Require(t, err)

}
