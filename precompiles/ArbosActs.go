//
// Copyright 2022, Offchain Labs, Inc. All rights reserved.
//

package precompiles

// This precompile represents ArbOS's internal actions as calls it makes to itself
type ArbosActs struct {
	Address addr // 0xa4b05

	CallerNotArbOSError func() error
}

func (con ArbosActs) StartBlock(c ctx, evm mech, l1BaseFee, l2BaseFeeLastBlock huge, l1BlockNumber, timeLastBlock uint64) error {
	return con.CallerNotArbOSError()
}

func (con ArbosActs) BatchPostingReport(c ctx, evm mech, batchPosterAddress addr, batchNumber, l1BaseFeeWei huge) error {
	return con.CallerNotArbOSError()
}
