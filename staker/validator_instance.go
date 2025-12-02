package staker

import (
	"context"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/validator"
	"github.com/offchainlabs/nitro/validator/client/redis"
	"github.com/offchainlabs/nitro/validator/inputs"
)

type LegacyValidatorInstance interface {
	CheckLegacyValid() error
	LegacyLastValidatedInfo() (*legacyLastBlockValidatedDbInfo, error)
	SetLegacyValidInfo(info *legacyLastBlockValidatedDbInfo)
}

// ValidatorInstance is an interface used by BlockValidator to carry out block and MEL-state validations. BlockValidatorInstance implements
// this interface to carry out block validations and MELValidatorInstance (TODO) implements this to carry out MEL-state validation
type ValidatorInstance interface {
	// Methods corresponding to ValidatorInstance's overall context
	IsValidatedGlobalStateNew(gs validator.GoGlobalState) bool
	LastValidatedGlobalState() validator.GoGlobalState
	SetLastValidatedGlobalState(gs validator.GoGlobalState)
	LastValidatedCount() (count uint64, isChainCaughtupToLastValidateCount bool, err error)
	LatestProcessedMessageCount() (uint64, error)                                       // latest processed but not validated message count
	CountAtValidatedGlobalState(gs validator.GoGlobalState) int64                       // the last validated message count correspoding to a given validated global state
	PositionsAtCount(count uint64) (beforePos, AfterPos GlobalStatePosition, err error) // returns the globalState position before and after processing message at the specified count
	CurrentGlobalState() validator.GoGlobalState                                        // last global state for which a validation entry was created
	ResetContextByCount(count uint64) error                                             // resets ValidatorInstance context to the context corresponding to the given message count
	ResetContextByGlobalStateAndCount(gs validator.GoGlobalState, count uint64)         // resets ValidatorInstance context to the context corresponding to given global state and message count
	ResetCaches()

	// Methods exposing validation specific components needed by ValidationCoordinator
	LatestWasmModuleRoot() common.Hash
	ExecSpawners() []validator.ExecutionSpawner
	ValidatorInputsWriter() (*inputs.Writer, error)
	RedisValidator() *redis.ValidationClient

	// Methods corresponding to ValidationEntry
	CanCreateValidationEntry(position uint64) (bool, error)
	CreateValidationEntry(ctx context.Context, position uint64) (*validationEntry, error)
	RecordEntry(ctx context.Context, entry *validationEntry) error

	// Methods to read and write GlobalStateValidatedInfo to database
	ReadLastValidatedInfo() (*GlobalStateValidatedInfo, error)
	WriteLastValidatedInfo(info GlobalStateValidatedInfo) error
}
