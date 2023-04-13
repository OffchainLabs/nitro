package server_common

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/validator"
)

type ValRun struct {
	containers.Promise[validator.GoGlobalState]
	root common.Hash
}

func (r *ValRun) WasmModuleRoot() common.Hash {
	return r.root
}

func NewValRun(root common.Hash) *ValRun {
	return &ValRun{
		Promise: containers.NewPromise[validator.GoGlobalState](nil),
		root:    root,
	}
}

func (r *ValRun) ConsumeResult(res validator.GoGlobalState, err error) {
	if err != nil {
		r.ProduceError(err)
	} else {
		r.Produce(res)
	}
}
