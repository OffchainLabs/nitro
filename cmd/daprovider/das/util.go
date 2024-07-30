// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/cmd/daprovider/das/dasutil"
	"github.com/offchainlabs/nitro/util/pretty"
)

func logPut(store string, data []byte, timeout uint64, reader dasutil.DASReader, more ...interface{}) {
	if len(more) == 0 {
		log.Trace(
			store, "message", pretty.FirstFewBytes(data), "timeout", time.Unix(int64(timeout), 0),
			"this", reader,
		)
	} else {
		log.Trace(
			store, "message", pretty.FirstFewBytes(data), "timeout", time.Unix(int64(timeout), 0),
			"this", reader, more,
		)
	}
}
