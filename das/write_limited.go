// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"context"
	"github.com/ethereum/go-ethereum/common"
)

type writeLimitedDataAvailabilityService struct {
	DataAvailabilityWriter
}

func NewWriteLimitedDataAvailabilityService(da DataAvailabilityWriter) *writeLimitedDataAvailabilityService {
	return &writeLimitedDataAvailabilityService{da}
}

func (*writeLimitedDataAvailabilityService) GetByHash(ctx context.Context, hash common.Hash) ([]byte, error) {
	panic("Logic error: writeLimitedDataAvailabilityService.GetByHash shouldn't be called.")
}
