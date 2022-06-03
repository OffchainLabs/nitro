// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"context"

	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/util/pretty"
)

type RedundantSimpleDASReader struct {
	inners []arbstate.DataAvailabilityReader
}

func NewRedundantSimpleDASReader(inners []arbstate.DataAvailabilityReader) arbstate.DataAvailabilityReader {
	return &RedundantSimpleDASReader{inners}
}

type rsdrResponse struct {
	data []byte
	err  error
}

func (r RedundantSimpleDASReader) GetByHash(ctx context.Context, hash []byte) ([]byte, error) {
	log.Trace("das.RedundantSimpleDASReader.GetByHash", "key", pretty.FirstFewBytes(hash), "this", r)

	subCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	numPending := len(r.inners)
	results := make(chan rsdrResponse, numPending)
	for _, inner := range r.inners {
		go func(inn arbstate.DataAvailabilityReader) {
			res, err := inn.GetByHash(subCtx, hash)
			results <- rsdrResponse{res, err}
		}(inner)
	}
	var anyError error
	for numPending > 0 {
		select {
		case res := <-results:
			if res.err != nil {
				anyError = res.err
				numPending--
			} else {
				return res.data, nil
			}
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return nil, anyError
}

func (r RedundantSimpleDASReader) HealthCheck(ctx context.Context) error {
	for _, simpleDASReader := range r.inners {
		err := simpleDASReader.HealthCheck(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}
