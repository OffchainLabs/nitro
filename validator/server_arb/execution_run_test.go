package server_arb

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/validator"
)

type mockMachine struct {
	gs         validator.GoGlobalState
	totalSteps uint64
}

func (m *mockMachine) Hash() common.Hash {
	if m.gs.PosInBatch == m.totalSteps-1 {
		return machineFinishedHash(m.gs)
	}
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
	mm := &mockMachine{}
	e := &executionRun{}
	ctx := context.Background()

	t.Run("basic argument checks", func(t *testing.T) {
		machStartIndex := uint64(0)
		stepSize := uint64(0)
		numRequiredHashes := uint64(0)
		_, err := e.machineHashesWithStepSize(ctx, machStartIndex, stepSize, numRequiredHashes)
		if !strings.Contains(err.Error(), "step size cannot be 0") {
			t.Error("Wrong error")
		}
		stepSize = uint64(1)
		_, err = e.machineHashesWithStepSize(ctx, machStartIndex, stepSize, numRequiredHashes)
		if !strings.Contains(err.Error(), "required number of hashes cannot be 0") {
			t.Error("Wrong error")
		}
	})
	t.Run("machine at start index 0 hash is the finished state hash", func(t *testing.T) {
		mm.gs = validator.GoGlobalState{
			Batch: 1,
		}
		machStartIndex := uint64(0)
		stepSize := uint64(1)
		numRequiredHashes := uint64(1)
		e.cache = &MachineCache{
			buildingLock: make(chan struct{}, 1),
			machines:     []MachineInterface{mm},
			finalMachine: mm,
		}
		go func() {
			<-time.After(time.Millisecond * 50)
			e.cache.buildingLock <- struct{}{}
		}()
		hashes, err := e.machineHashesWithStepSize(ctx, machStartIndex, stepSize, numRequiredHashes)
		if err != nil {
			t.Fatal(err)
		}
		expected := machineFinishedHash(mm.gs)
		if len(hashes) != 1 {
			t.Error("Wanted one hash")
		}
		if expected != hashes[0] {
			t.Errorf("Wanted %#x, got %#x", expected, hashes[0])
		}
	})
	t.Run("can step in step size increments and collect hashes", func(t *testing.T) {
		initialGs := validator.GoGlobalState{
			Batch:      1,
			PosInBatch: 0,
		}
		mm.gs = initialGs
		mm.totalSteps = 20
		machStartIndex := uint64(0)
		stepSize := uint64(5)
		numRequiredHashes := uint64(4)
		e.cache = &MachineCache{
			buildingLock: make(chan struct{}, 1),
			machines:     []MachineInterface{mm},
			finalMachine: mm,
		}
		go func() {
			<-time.After(time.Millisecond * 50)
			e.cache.buildingLock <- struct{}{}
		}()
		hashes, err := e.machineHashesWithStepSize(ctx, machStartIndex, stepSize, numRequiredHashes)
		if err != nil {
			t.Fatal(err)
		}
		expectedHashes := make([]common.Hash, 0)
		for i := uint64(0); i < 4; i++ {
			if i == 0 {
				expectedHashes = append(expectedHashes, machineFinishedHash(initialGs))
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
		initialGs := validator.GoGlobalState{
			Batch:      1,
			PosInBatch: 0,
		}
		mm.gs = initialGs
		mm.totalSteps = 20
		machStartIndex := uint64(0)
		stepSize := uint64(5)
		numRequiredHashes := uint64(10)
		e.cache = &MachineCache{
			buildingLock: make(chan struct{}, 1),
			machines:     []MachineInterface{mm},
			finalMachine: mm,
		}
		go func() {
			<-time.After(time.Millisecond * 50)
			e.cache.buildingLock <- struct{}{}
		}()
		hashes, err := e.machineHashesWithStepSize(ctx, machStartIndex, stepSize, numRequiredHashes)
		if err != nil {
			t.Fatal(err)
		}
		expectedHashes := make([]common.Hash, 0)
		for i := uint64(0); i < 4; i++ {
			if i == 0 {
				expectedHashes = append(expectedHashes, machineFinishedHash(initialGs))
				continue
			}
			gs := validator.GoGlobalState{
				Batch:      1,
				PosInBatch: uint64(i * stepSize),
			}
			expectedHashes = append(expectedHashes, gs.Hash())
		}
		if len(hashes) >= int(numRequiredHashes) {
			t.Fatal("Wanted fewer hashes than the max iterations")
		}
		for i := range hashes {
			if expectedHashes[i] != hashes[i] {
				t.Errorf("Wanted at index %d, %#x, got %#x", i, expectedHashes[i], hashes[i])
			}
		}
	})
}
