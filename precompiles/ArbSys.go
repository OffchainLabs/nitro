// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package precompiles

import (
	"errors"
	"math/big"

	"github.com/offchainlabs/nitro/arbos/l2pricing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/merkletree"
)

// Provides system-level functionality for interacting with L1 and understanding the call stack.
type ArbSys struct {
	Address                 addr
	L2ToL1Tx                func(ctx, mech, addr, addr, huge, huge, huge, huge, huge, huge, []byte) error
	L2ToL1TxGasCost         func(addr, addr, huge, huge, huge, huge, huge, huge, []byte) (uint64, error)
	SendMerkleUpdate        func(ctx, mech, huge, bytes32, huge) error
	SendMerkleUpdateGasCost func(huge, bytes32, huge) (uint64, error)

	// deprecated event
	L2ToL1Transaction        func(ctx, mech, addr, addr, huge, huge, huge, huge, huge, huge, huge, []byte) error
	L2ToL1TransactionGasCost func(addr, addr, huge, huge, huge, huge, huge, huge, huge, []byte) (uint64, error)
}

var InvalidBlockNum = errors.New("Invalid block number")

func (con *ArbSys) emitL2ToL1Tx(
	c ctx,
	evm mech,
	destination addr,
	hash huge,
	position huge,
	ethBlockNum huge,
	callvalue huge,
	data []byte,
) error {
	if c.State.FormatVersion() >= l2pricing.FirstExponentialPricingVersion {
		return con.L2ToL1Tx(
			c,
			evm,
			c.caller,
			destination,
			hash,
			position,
			evm.Context.BlockNumber,
			ethBlockNum,
			evm.Context.Time,
			callvalue,
			data,
		)
	} else {
		return con.L2ToL1Transaction(
			c,
			evm,
			c.caller,
			destination,
			hash,
			position,
			big.NewInt(0),
			evm.Context.BlockNumber,
			ethBlockNum,
			evm.Context.Time,
			callvalue,
			data,
		)
	}
}

// Gets the current L2 block number
func (con *ArbSys) ArbBlockNumber(c ctx, evm mech) (huge, error) {
	return evm.Context.BlockNumber, nil
}

// Gets the L2 block hash, if sufficiently recent
func (con *ArbSys) ArbBlockHash(c ctx, evm mech, arbBlockNumber *big.Int) (bytes32, error) {
	if !arbBlockNumber.IsUint64() {
		return bytes32{}, InvalidBlockNum
	}
	requestedBlockNum := arbBlockNumber.Uint64()

	currentNumber := evm.Context.BlockNumber.Uint64()
	if requestedBlockNum >= currentNumber || requestedBlockNum+256 < currentNumber {
		return common.Hash{}, errors.New("invalid block number for ArbBlockHAsh")
	}

	return evm.Context.GetHash(requestedBlockNum), nil
}

// Gets the rollup's unique chain identifier
func (con *ArbSys) ArbChainID(c ctx, evm mech) (huge, error) {
	return evm.ChainConfig().ChainID, nil
}

// Gets the current ArbOS version
func (con *ArbSys) ArbOSVersion(c ctx, evm mech) (huge, error) {
	version := new(big.Int).SetUint64(55 + c.State.FormatVersion()) // Nitro starts at version 56
	return version, nil
}

// Returns 0 since Nitro has no concept of storage gas
func (con *ArbSys) GetStorageGasAvailable(c ctx, evm mech) (huge, error) {
	return big.NewInt(0), nil
}

// Checks if the call is top-level
func (con *ArbSys) IsTopLevelCall(c ctx, evm mech) (bool, error) {
	return evm.Depth() <= 2, nil
}

// Gets the contract's L2 alias
func (con *ArbSys) MapL1SenderContractAddressToL2Alias(c ctx, sender addr, dest addr) (addr, error) {
	return util.RemapL1Address(sender), nil
}

// Checks if the caller's caller was aliased
func (con *ArbSys) WasMyCallersAddressAliased(c ctx, evm mech) (bool, error) {
	aliased := evm.Depth() == 2 && util.DoesTxTypeAlias(c.txProcessor.TopTxType)
	return aliased, nil
}

// Gets the caller's caller without any potential aliasing
func (con *ArbSys) MyCallersAddressWithoutAliasing(c ctx, evm mech) (addr, error) {

	address := addr{}

	if evm.Depth() > 1 {
		address = c.txProcessor.Callers[evm.Depth()-2]
	}

	if evm.Depth() == 2 && util.DoesTxTypeAlias(c.txProcessor.TopTxType) {
		address = util.InverseRemapL1Address(address)
	}

	return address, nil
}

// Sends a transaction to L1, adding it to the outbox
func (con *ArbSys) SendTxToL1(c ctx, evm mech, value huge, destination addr, calldataForL1 []byte) (huge, error) {
	l1BlockNum, err := c.txProcessor.L1BlockNumber(vm.BlockContext{})
	if err != nil {
		return nil, err
	}
	bigL1BlockNum := arbmath.UintToBig(l1BlockNum)

	arbosState := c.State
	sendHash, err := arbosState.KeccakHash(
		c.caller.Bytes(),
		destination.Bytes(),
		math.U256Bytes(evm.Context.BlockNumber),
		math.U256Bytes(bigL1BlockNum),
		math.U256Bytes(evm.Context.Time),
		common.BigToHash(value).Bytes(),
		calldataForL1,
	)
	if err != nil {
		return nil, err
	}
	merkleAcc := arbosState.SendMerkleAccumulator()
	merkleUpdateEvents, err := merkleAcc.Append(sendHash)
	if err != nil {
		return nil, err
	}

	size, err := merkleAcc.Size()
	if err != nil {
		return nil, err
	}

	// burn the callvalue, which was previously deposited to this precompile's account
	if err := util.BurnBalance(&con.Address, value, evm, util.TracingDuringEVM, "withdraw"); err != nil {
		return nil, err
	}

	for _, merkleUpdateEvent := range merkleUpdateEvents {
		position := merkletree.LevelAndLeaf{
			Level: merkleUpdateEvent.Level,
			Leaf:  merkleUpdateEvent.NumLeaves,
		}
		err := con.SendMerkleUpdate(
			c,
			evm,
			big.NewInt(0),
			merkleUpdateEvent.Hash,
			position.ToBigInt(),
		)
		if err != nil {
			return nil, err
		}
	}

	leafNum := big.NewInt(int64(size - 1))

	err = con.emitL2ToL1Tx(
		c,
		evm,
		destination,
		sendHash.Big(),
		leafNum,
		bigL1BlockNum,
		value,
		calldataForL1,
	)

	return sendHash.Big(), err
}

// Gets the root, size, and partials of the outbox Merkle tree state (caller must be the 0 address)
func (con ArbSys) SendMerkleTreeState(c ctx, evm mech) (huge, bytes32, []bytes32, error) {
	if c.caller != (addr{}) {
		return nil, bytes32{}, nil, errors.New("method can only be called by address zero")
	}

	// OK to not charge gas, because method is only callable by address zero

	size, rootHash, rawPartials, _ := c.State.SendMerkleAccumulator().StateForExport()
	partials := make([]bytes32, len(rawPartials))
	for i, par := range rawPartials {
		partials[i] = bytes32(par)
	}
	return big.NewInt(int64(size)), bytes32(rootHash), partials, nil
}

// Send paid eth to the destination on L1
func (con ArbSys) WithdrawEth(c ctx, evm mech, value huge, destination addr) (huge, error) {
	return con.SendTxToL1(c, evm, value, destination, []byte{})
}
