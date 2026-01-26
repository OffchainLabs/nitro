// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package server_arb

/*
#cgo CFLAGS: -g -I../../target/include/
#include "arbitrator.h"

extern ResolvedPreimage preimageResolver(size_t context, uint8_t preimageType, const uint8_t* hash);

ResolvedPreimage preimageResolverC(size_t context, uint8_t preimageType, const uint8_t* hash) {
  return preimageResolver(context, preimageType, hash);
}

extern ResolvedPreimage mapResolver(size_t context, const uint8_t* hash);

ResolvedPreimage mapResolverC(size_t context, const uint8_t* hash) {
  return mapResolver(context, hash);
}
*/
import "C"
