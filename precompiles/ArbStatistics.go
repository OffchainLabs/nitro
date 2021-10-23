//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package precompiles

import (
	"errors"
)

type ArbStatistics struct{}

func (con ArbStatistics) GetStats(caller addr, st *stateDB) (huge, huge, huge, huge, huge, huge, error) {
	return nil, nil, nil, nil, nil, nil, errors.New("unimplemented")
}

func (con ArbStatistics) GetStatsGasCost() uint64 {
	return 0
}
