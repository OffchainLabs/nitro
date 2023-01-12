package server_api

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/validator"
)

const Namespace string = "validation"

type ValidationServerAPI struct {
	spawner validator.ValidationSpawner
}

func (a *ValidationServerAPI) Name() string {
	return a.spawner.Name()
}

func (a *ValidationServerAPI) Room() int {
	return a.spawner.Room()
}

func (a *ValidationServerAPI) Validate(ctx context.Context, entry *ValidationInputJson, moduleRoot common.Hash) (validator.GoGlobalState, error) {
	valInput, err := ValidationInputFromJson(entry)
	if err != nil {
		return validator.GoGlobalState{}, err
	}
	valRun := a.spawner.Launch(valInput, moduleRoot)
	err = valRun.WaitReady(ctx)
	if err != nil {
		return validator.GoGlobalState{}, err
	}
	return valRun.Result()
}

func NewValidationServerAPI(spawner validator.ValidationSpawner) *ValidationServerAPI {
	return &ValidationServerAPI{spawner}
}
