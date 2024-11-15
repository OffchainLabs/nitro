package server_common

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/validator"
)

type ValRun struct {
	containers.PromiseInterface[validator.GoGlobalState]
	root common.Hash
}

func (r *ValRun) WasmModuleRoot() common.Hash {
	return r.root
}

func NewValRun(promise containers.PromiseInterface[validator.GoGlobalState], root common.Hash) *ValRun {
	return &ValRun{
		PromiseInterface: promise,
		root:             root,
	}
}
