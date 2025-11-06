// Copyright 2021-2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package daprovider

import (
	"errors"

	"github.com/offchainlabs/nitro/util/containers"
)

// ErrFallbackRequested is returned by a DA provider to explicitly signal that
// the batch poster should fall back to the next available DA writer.
// Without this explicit signal, any error will cause batch posting to fail
// rather than automatically falling back, preventing expensive surprise costs
// from fixable infrastructure issues. Although the rpcclient will retry a certain
// number of times on transient errors, there could be other issues like
// misconfigurations or temporary outages that are better fixed by operator
// intervention than automatically falling back.
var ErrFallbackRequested = errors.New("DA provider requests fallback to next writer")

type Writer interface {
	// Store posts the batch data to the invoking DA provider
	// And returns sequencerMsg which is later used to retrieve the batch data
	Store(
		message []byte,
		timeout uint64,
	) containers.PromiseInterface[[]byte]
}
