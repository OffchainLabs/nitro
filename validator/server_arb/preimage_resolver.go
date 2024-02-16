// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package server_arb

/*
#cgo CFLAGS: -g -Wall -I../../target/include/
#include "arbitrator.h"

extern ResolvedPreimage preimageResolver(size_t context, uint8_t preimageType, const uint8_t* hash);

ResolvedPreimage preimageResolverC(size_t context, uint8_t preimageType, const uint8_t* hash) {
  return preimageResolver(context, preimageType, hash);
}
*/
import "C"
