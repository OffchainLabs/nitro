// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package anytrust

import (
	"time"

	"github.com/ethereum/go-ethereum/log"

	anytrustutil "github.com/offchainlabs/nitro/daprovider/anytrust/util"
	"github.com/offchainlabs/nitro/util/pretty"
)

func logPut(store string, data []byte, timeout uint64, reader anytrustutil.DASReader, more ...interface{}) {
	if len(more) == 0 {
		// #nosec G115
		log.Trace(
			store, "message", pretty.FirstFewBytes(data), "timeout", time.Unix(int64(timeout), 0),
			"this", reader,
		)
	} else {
		// #nosec G115
		log.Trace(
			store, "message", pretty.FirstFewBytes(data), "timeout", time.Unix(int64(timeout), 0),
			"this", reader, more,
		)
	}
}
