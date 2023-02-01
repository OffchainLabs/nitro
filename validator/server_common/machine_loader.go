package server_common

import (
	"context"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/util/readymarker"
)

type MachineStatus[M any] struct {
	readymarker.ReadyMarker
	Machine *M
}

func newMachineStatus[M any]() *MachineStatus[M] {
	return &MachineStatus[M]{
		ReadyMarker: readymarker.NewReadyMarker(),
	}
}

type MachineLoader[M any] struct {
	mapMutex      sync.Mutex
	machines      map[common.Hash]*MachineStatus[M]
	locator       *MachineLocator
	createMachine func(ctx context.Context, moduleRoot common.Hash) (*M, error)
}

func NewMachineLoader[M any](
	locator *MachineLocator,
	createMachine func(ctx context.Context, moduleRoot common.Hash) (*M, error),
) *MachineLoader[M] {

	return &MachineLoader[M]{
		machines:      make(map[common.Hash]*MachineStatus[M]),
		locator:       locator,
		createMachine: createMachine,
	}
}

func (l *MachineLoader[M]) GetMachine(ctx context.Context, moduleRoot common.Hash) (*M, error) {
	if moduleRoot == (common.Hash{}) {
		moduleRoot = l.locator.LatestWasmModuleRoot()
		if (moduleRoot == common.Hash{}) {
			return nil, ErrMachineNotFound
		}
	}
	l.mapMutex.Lock()
	status := l.machines[moduleRoot]
	if status == nil {
		status = newMachineStatus[M]()
		l.machines[moduleRoot] = status
		go func() {
			machine, err := l.createMachine(context.Background(), moduleRoot)
			if err == nil {
				status.Machine = machine
			}
			status.SignalReady(err)
		}()
	}
	l.mapMutex.Unlock()
	err := status.WaitReady(ctx)
	if err != nil {
		return nil, err
	}
	return status.Machine, nil
}

func (l *MachineLoader[M]) ForEachMachine(runme func(*M) error) error {
	for _, stat := range l.machines {
		if stat.Machine != nil {
			if err := runme(stat.Machine); err != nil {
				return err
			}
		}
	}
	return nil
}
