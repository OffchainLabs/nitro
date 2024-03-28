package server_arb

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/validator/server_common"
)

type ArbitratorMachineConfig struct {
	WavmBinaryPath       string
	UntilHostIoStatePath string
}

var DefaultArbitratorMachineConfig = ArbitratorMachineConfig{
	WavmBinaryPath:       "machine.wavm.br",
	UntilHostIoStatePath: "until-host-io-state.bin",
}

type arbMachines struct {
	zeroStep *ArbitratorMachine
	hostIo   *ArbitratorMachine
}

type ArbMachineLoader struct {
	server_common.MachineLoader[arbMachines]
}

func NewArbMachineLoader(config *ArbitratorMachineConfig, locator *server_common.MachineLocator) *ArbMachineLoader {
	createMachineFunc := func(ctx context.Context, moduleRoot common.Hash, opts ...server_common.MachineLoaderOpt) (*arbMachines, error) {
		return createArbMachine(ctx, locator, config, moduleRoot, opts...)
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

func (a *ArbMachineLoader) GetZeroStepMachine(ctx context.Context, moduleRoot common.Hash, opts ...server_common.MachineLoaderOpt) (*ArbitratorMachine, error) {
	machines, err := a.GetMachine(ctx, moduleRoot, opts...)
	if err != nil {
		return nil, err
	}
	return machines.zeroStep, nil
}
