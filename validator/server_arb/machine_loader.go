// Copyright 2023-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package server_arb

import (
	"context"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/validator/server_common"
)

type ArbitratorMachineConfig struct {
	WavmBinaryPath       string `koanf:"wavm-binary-path" reload:"hot"`
	UntilHostIoStatePath string `koanf:"util-host-io-state-path" reload:"hot"`
}

var DefaultArbitratorMachineConfig = ArbitratorMachineConfig{
	WavmBinaryPath:       "unified_machine.wavm.br",
	UntilHostIoStatePath: "unified-until-host-io-state.bin",
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
