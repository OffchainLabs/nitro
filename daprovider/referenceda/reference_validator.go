// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package referenceda

import (
	"context"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/daprovider"
)

type Validator struct {
	storage *InMemoryStorage
}

func NewValidator() *Validator {
	return &Validator{
		storage: GetInMemoryStorage(),
	}
}

func (v *Validator) RecordPreimages(ctx context.Context, batch []byte) ([]daprovider.PreimageWithType, error) {
	panic("not implemented yet")
}

func (v *Validator) GenerateProof(ctx context.Context, preimageType arbutil.PreimageType, hash common.Hash, offset uint64) ([]byte, error) {
	panic("not implemented yet")
}
