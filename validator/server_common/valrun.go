package server_common

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/util/readymarker"
	"github.com/offchainlabs/nitro/validator"
)

type ValRun struct {
	readymarker.ReadyMarker
	root   common.Hash
	result validator.GoGlobalState
}

func (r *ValRun) Result() (validator.GoGlobalState, error) {
	if err := r.TestReady(); err != nil {
		return validator.GoGlobalState{}, err
	}
	return r.result, nil
}

func (r *ValRun) WasmModuleRoot() common.Hash {
	return r.root
}

func (r *ValRun) Close() {}

func NewValRun(root common.Hash) *ValRun {
	return &ValRun{
		ReadyMarker: readymarker.NewReadyMarker(),
		root:        root,
	}
}

func (r *ValRun) ConsumeResult(res validator.GoGlobalState, err error) {
	r.result = res
	r.SignalReady(err)
}
