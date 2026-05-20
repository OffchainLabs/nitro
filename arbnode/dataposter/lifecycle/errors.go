// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package lifecycle

import (
	"errors"
)

var ErrExceedsMaxMempoolSize = errors.New("posting this transaction will exceed max mempool size")
