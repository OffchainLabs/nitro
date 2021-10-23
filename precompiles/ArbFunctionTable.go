//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package precompiles

import (
	"errors"
)

type ArbFunctionTable struct{}

func (con ArbFunctionTable) Get(caller addr, st *stateDB, addr addr, index huge) (huge, bool, huge, error) {
	return nil, false, nil, errors.New("unimplemented")
}

func (con ArbFunctionTable) GetGasCost(addr addr, index huge) uint64 {
	return 0
}

func (con ArbFunctionTable) Size(caller addr, st *stateDB, addr addr) (huge, error) {
	return nil, errors.New("unimplemented")
}

func (con ArbFunctionTable) SizeGasCost(addr addr) uint64 {
	return 0
}

func (con ArbFunctionTable) Upload(caller addr, st *stateDB, buf []byte) error {
	return errors.New("unimplemented")
}

func (con ArbFunctionTable) UploadGasCost(buf []byte) uint64 {
	return 0
}
