// Copyright 2023-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package server_arb

import (
	"context"

	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/validator/server_common"
)

type ArbitratorMachineConfig struct {
	WavmBinaryPath       string `koanf:"wavm-binary-path" reload:"hot"`
	UntilHostIoStatePath string `koanf:"until-host-io-state-path" reload:"hot"`
}

var DefaultArbitratorMachineConfig = ArbitratorMachineConfig{
	WavmBinaryPath:       "machine.wavm.br",
	UntilHostIoStatePath: "until-host-io-state.bin",
}

func ArbitratorMachineConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.String(prefix+".wavm-binary-path", DefaultArbitratorMachineConfig.WavmBinaryPath, "path to the machine's wavm binary relative to machine path")
	f.String(prefix+".until-host-io-state-path", DefaultArbitratorMachineConfig.UntilHostIoStatePath, "path to the machine's until-host-io state file relative to machine path")
}

type arbMachines struct {
	zeroStep *ArbitratorMachine
	hostIo   *ArbitratorMachine
}

type ArbMachineLoader struct {
	server_common.MachineLoader[arbMachines]
}

func NewArbMachineLoader(config *ArbitratorMachineConfig, locator *server_common.MachineLocator) *ArbMachineLoader {
	createMachineFunc := func(ctx context.Context, moduleRoot common.Hash) (*arbMachines, error) {
		return createArbMachine(ctx, locator, config, moduleRoot)
	}
	return &ArbMachineLoader{
		MachineLoader: *server_common.NewMachineLoader[arbMachines](locator, createMachineFunc),
	}
}

func (a *ArbMachineLoader) GetHostIoMachine(ctx context.Context, moduleRoot common.Hash) (*ArbitratorMachine, error) {
	machines, err := a.GetMachine(ctx, moduleRoot)
	if err != nil {
		return nil, err
	}
	return machines.hostIo, nil
}

func (a *ArbMachineLoader) GetZeroStepMachine(ctx context.Context, moduleRoot common.Hash) (*ArbitratorMachine, error) {
	machines, err := a.GetMachine(ctx, moduleRoot)
	if err != nil {
		return nil, err
	}
	return machines.zeroStep, nil
}
