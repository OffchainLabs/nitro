// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package metrics

type Counter interface{ Inc(int64) }
type Gauge interface{ Update(int64) }

func NewRegisteredCounter(name string, r interface{}) Counter  { return nil }
func NewRegisteredGauge(name string, r interface{}) Gauge      { return nil }
