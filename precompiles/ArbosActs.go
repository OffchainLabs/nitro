//
// Copyright 2022, Offchain Labs, Inc. All rights reserved.
//

package precompiles

// ArbosActs precompile represents ArbOS's internal actions as calls it makes to itself
type ArbosActs struct {
	Address addr // 0xa4b05

	CallerNotArbOSError func() error
}

func (con ArbosActs) StartBlock(c ctx, evm mech, l1BaseFee huge, l1BlockNumber, l2BlockNumber, timeLastBlock uint64) error {
	return con.CallerNotArbOSError()
}

func (con ArbosActs) BatchPostingReport(c ctx, evm mech, batchTimestamp huge, batchPosterAddress addr, batchNumber uint64, batchDataGas uint64, l1BaseFeeWei huge) error {
	return con.CallerNotArbOSError()
}
