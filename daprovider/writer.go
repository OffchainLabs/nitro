// Copyright 2021-2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package daprovider

import (
	"github.com/offchainlabs/nitro/util/containers"
)

type Writer interface {
	// Store posts the batch data to the invoking DA provider
	// And returns sequencerMsg which is later used to retrieve the batch data
	Store(
		message []byte,
		timeout uint64,
	) containers.PromiseInterface[[]byte]
}
