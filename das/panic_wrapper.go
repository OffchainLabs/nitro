// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbstate"
)

type PanicWrapper struct {
	DataAvailabilityService
}

func NewPanicWrapper(dataAvailabilityService DataAvailabilityService) DataAvailabilityService {
	return &PanicWrapper{
		DataAvailabilityService: dataAvailabilityService,
	}
}

func (w *PanicWrapper) GetByHash(ctx context.Context, hash common.Hash) ([]byte, error) {
	data, err := w.DataAvailabilityService.GetByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			log.Error("DAS hash lookup failed from cancelled context")
			return nil, err
		}
		panic(fmt.Sprintf("panic wrapper GetByHash: %v", err))
	}
	return data, nil
}

func (w *PanicWrapper) Store(ctx context.Context, message []byte, timeout uint64, sig []byte) (*arbstate.DataAvailabilityCertificate, error) {
	cert, err := w.DataAvailabilityService.Store(ctx, message, timeout, sig)
	if err != nil {
		panic(fmt.Sprintf("panic wrapper Store: %v", err))
	}
	return cert, nil
}

func (w *PanicWrapper) String() string {
	return fmt.Sprintf("PanicWrapper{%v}", w.DataAvailabilityService)
}
