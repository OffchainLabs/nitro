package server_arb

import (
	"context"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/validator"
)

type mockMachine struct {
	gs         validator.GoGlobalState
	totalSteps uint64
}

func (m *mockMachine) Hash() common.Hash {
	return m.gs.Hash()
}

func (m *mockMachine) GetGlobalState() validator.GoGlobalState {
	return m.gs
}

func (m *mockMachine) Step(ctx context.Context, stepSize uint64) error {
	for i := uint64(0); i < stepSize; i++ {
		if m.gs.PosInBatch == m.totalSteps-1 {
			return nil
		}
		m.gs.PosInBatch += 1
	}
	return nil
}

func (m *mockMachine) CloneMachineInterface() MachineInterface {
	return &mockMachine{
		gs:         validator.GoGlobalState{Batch: m.gs.Batch, PosInBatch: m.gs.PosInBatch},
		totalSteps: m.totalSteps,
	}
}
func (m *mockMachine) GetStepCount() uint64 {
	return 0
}
func (m *mockMachine) IsRunning() bool {
	return m.gs.PosInBatch < m.totalSteps-1
}
func (m *mockMachine) IsErrored() bool {
	return false
}
func (m *mockMachine) ValidForStep(uint64) bool {
	return true
}
func (m *mockMachine) Status() uint8 {
	if m.gs.PosInBatch == m.totalSteps-1 {
		return uint8(validator.MachineStatusFinished)
	}
	return uint8(validator.MachineStatusRunning)
}
func (m *mockMachine) ProveNextStep() []byte {
	return nil
}
func (m *mockMachine) Freeze()  {}
func (m *mockMachine) Destroy() {}

func Test_machineHashesWithStep(t *testing.T) {
	t.Run("basic argument checks", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		e := &executionRun{}
		machStartIndex := uint64(0)
		stepSize := uint64(0)
		maxIterations := uint64(0)
		_, err := e.machineHashesWithStepSize(ctx, machStartIndex, stepSize, maxIterations)
		if err == nil || !strings.Contains(err.Error(), "step size cannot be 0") {
			t.Error("Wrong error")
		}
		stepSize = uint64(1)
		_, err = e.machineHashesWithStepSize(ctx, machStartIndex, stepSize, maxIterations)
		if err == nil || !strings.Contains(err.Error(), "number of iterations cannot be 0") {
			t.Error("Wrong error")
		}
	})
	t.Run("machine at start index 0 hash is the finished state hash", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		mm := &mockMachine{
			gs: validator.GoGlobalState{
				Batch: 1,
			},
			totalSteps: 20,
		}
		machStartIndex := uint64(0)
		stepSize := uint64(1)
		maxIterations := uint64(1)
		e := &executionRun{
			cache: NewMachineCache(ctx, func(_ context.Context) (MachineInterface, error) {
				return mm, nil
			}, &DefaultMachineCacheConfig),
		}

		hashes, err := e.machineHashesWithStepSize(ctx, machStartIndex, stepSize, maxIterations)
		if err != nil {
			t.Fatal(err)
		}
		expected := mm.gs.Hash()
		if len(hashes) != 1 {
			t.Error("Wanted one hash")
		}
		if expected != hashes[0] {
			t.Errorf("Wanted %#x, got %#x", expected, hashes[0])
		}
	})
	t.Run("can step in step size increments and collect hashes", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		initialGs := validator.GoGlobalState{
			Batch:      1,
			PosInBatch: 0,
		}
		mm := &mockMachine{
			gs:         initialGs,
			totalSteps: 20,
		}
		machStartIndex := uint64(0)
		stepSize := uint64(5)
		maxIterations := uint64(4)
		e := &executionRun{
			cache: NewMachineCache(ctx, func(_ context.Context) (MachineInterface, error) {
				return mm, nil
			}, &DefaultMachineCacheConfig),
		}
		hashes, err := e.machineHashesWithStepSize(ctx, machStartIndex, stepSize, maxIterations)
		if err != nil {
			t.Fatal(err)
		}
		expectedHashes := make([]common.Hash, 0)
		for i := uint64(0); i < 4; i++ {
			if i == 0 {
				expectedHashes = append(expectedHashes, initialGs.Hash())
				continue
			}
			gs := validator.GoGlobalState{
				Batch:      1,
				PosInBatch: uint64(i * stepSize),
			}
			expectedHashes = append(expectedHashes, gs.Hash())
		}
		if len(hashes) != len(expectedHashes) {
			t.Fatal("Wanted one hash")
		}
		for i := range hashes {
			if expectedHashes[i] != hashes[i] {
				t.Errorf("Wanted at index %d, %#x, got %#x", i, expectedHashes[i], hashes[i])
			}
		}
	})
	t.Run("if finishes execution early, can return a smaller number of hashes than the expected max iterations", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		initialGs := validator.GoGlobalState{
			Batch:      1,
			PosInBatch: 0,
		}
		mm := &mockMachine{
			gs:         initialGs,
			totalSteps: 20,
		}
		machStartIndex := uint64(0)
		stepSize := uint64(5)
		maxIterations := uint64(10)
		e := &executionRun{
			cache: NewMachineCache(ctx, func(_ context.Context) (MachineInterface, error) {
				return mm, nil
			}, &DefaultMachineCacheConfig),
		}

		hashes, err := e.machineHashesWithStepSize(ctx, machStartIndex, stepSize, maxIterations)
		if err != nil {
			t.Fatal(err)
		}
		expectedHashes := make([]common.Hash, 0)
		for i := uint64(0); i < 4; i++ {
			if i == 0 {
				expectedHashes = append(expectedHashes, initialGs.Hash())
				continue
			}
			gs := validator.GoGlobalState{
				Batch:      1,
				PosInBatch: uint64(i * stepSize),
			}
			expectedHashes = append(expectedHashes, gs.Hash())
		}
		expectedHashes = append(expectedHashes, validator.GoGlobalState{
			Batch:      1,
			PosInBatch: mm.totalSteps - 1,
		}.Hash())
		if uint64(len(hashes)) >= maxIterations {
			t.Fatal("Wanted fewer hashes than the max iterations")
		}
		for i := range hashes {
			if expectedHashes[i] != hashes[i] {
				t.Errorf("Wanted at index %d, %#x, got %#x", i, expectedHashes[i], hashes[i])
			}
		}
	})
}
