// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package precompiles

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/merkletree"
)

// ArbSys provides system-level functionality for interacting with L1 and understanding the call stack.
type ArbSys struct {
	Address                 addr // 0x64
	L2ToL1Tx                func(ctx, mech, addr, addr, huge, huge, huge, huge, huge, huge, []byte) error
	L2ToL1TxGasCost         func(addr, addr, huge, huge, huge, huge, huge, huge, []byte) (uint64, error)
	SendMerkleUpdate        func(ctx, mech, huge, bytes32, huge) error
	SendMerkleUpdateGasCost func(huge, bytes32, huge) (uint64, error)
	InvalidBlockNumberError func(huge, huge) error

	// deprecated event
	L2ToL1Transaction        func(ctx, mech, addr, addr, huge, huge, huge, huge, huge, huge, huge, []byte) error
	L2ToL1TransactionGasCost func(addr, addr, huge, huge, huge, huge, huge, huge, huge, []byte) (uint64, error)
}

// ArbBlockNumber gets the current L2 block number
func (con *ArbSys) ArbBlockNumber(c ctx, evm mech) (huge, error) {
	return evm.Context.BlockNumber, nil
}

// ArbBlockHash gets the L2 block hash, if sufficiently recent
func (con *ArbSys) ArbBlockHash(c ctx, evm mech, arbBlockNumber *big.Int) (bytes32, error) {
	if !arbBlockNumber.IsUint64() {
		if c.State.ArbOSVersion() >= 11 {
			return bytes32{}, con.InvalidBlockNumberError(arbBlockNumber, evm.Context.BlockNumber)
		}
		return bytes32{}, errors.New("invalid block number")
	}
	requestedBlockNum := arbBlockNumber.Uint64()

	currentNumber := evm.Context.BlockNumber.Uint64()
	if requestedBlockNum >= currentNumber || requestedBlockNum+256 < currentNumber {
		if c.State.ArbOSVersion() >= 11 {
			return common.Hash{}, con.InvalidBlockNumberError(arbBlockNumber, evm.Context.BlockNumber)
		}
		return common.Hash{}, errors.New("invalid block number for ArbBlockHAsh")
	}

	return evm.Context.GetHash(requestedBlockNum), nil
}

// ArbChainID gets the rollup's unique chain identifier
func (con *ArbSys) ArbChainID(c ctx, evm mech) (huge, error) {
	return evm.ChainConfig().ChainID, nil
}

// ArbOSVersion gets the current ArbOS version
func (con *ArbSys) ArbOSVersion(c ctx, evm mech) (huge, error) {
	version := new(big.Int).SetUint64(55 + c.State.ArbOSVersion()) // Nitro starts at version 56
	return version, nil
}

// GetStorageGasAvailable returns 0 since Nitro has no concept of storage gas
func (con *ArbSys) GetStorageGasAvailable(c ctx, evm mech) (huge, error) {
	return big.NewInt(0), nil
}

// IsTopLevelCall checks if the call is top-level (deprecated)
func (con *ArbSys) IsTopLevelCall(c ctx, evm mech) (bool, error) {
	return evm.Depth() <= 2, nil
}

// MapL1SenderContractAddressToL2Alias gets the contract's L2 alias
func (con *ArbSys) MapL1SenderContractAddressToL2Alias(c ctx, sender addr, dest addr) (addr, error) {
	return util.RemapL1Address(sender), nil
}

// WasMyCallersAddressAliased checks if the caller's caller was aliased
func (con *ArbSys) WasMyCallersAddressAliased(c ctx, evm mech) (bool, error) {
	topLevel := con.isTopLevel(c, evm)
	if c.State.ArbOSVersion() < 6 {
		topLevel = evm.Depth() == 2
	}
	aliased := topLevel && util.DoesTxTypeAlias(c.txProcessor.TopTxType)
	return aliased, nil
}

// MyCallersAddressWithoutAliasing gets the caller's caller without any potential aliasing
func (con *ArbSys) MyCallersAddressWithoutAliasing(c ctx, evm mech) (addr, error) {

	address := addr{}

	if evm.Depth() > 1 {
		address = c.txProcessor.Callers[evm.Depth()-2]
	}

	aliased, err := con.WasMyCallersAddressAliased(c, evm)
	if aliased {
		address = util.InverseRemapL1Address(address)
	}
	return address, err
}

// SendTxToL1 sends a transaction to L1, adding it to the outbox
func (con *ArbSys) SendTxToL1(c ctx, evm mech, value huge, destination addr, calldataForL1 []byte) (huge, error) {
	l1BlockNum, err := c.txProcessor.L1BlockNumber(vm.BlockContext{})
	if err != nil {
		return nil, err
	}
	bigL1BlockNum := arbmath.UintToBig(l1BlockNum)

	arbosState := c.State
	var t big.Int
	t.SetUint64(evm.Context.Time)
	sendHash, err := arbosState.KeccakHash(
		c.caller.Bytes(),
		destination.Bytes(),
		math.U256Bytes(evm.Context.BlockNumber),
		math.U256Bytes(bigL1BlockNum),
		math.U256Bytes(&t),
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

	var blockTime big.Int
	blockTime.SetUint64(evm.Context.Time)
	err = con.L2ToL1Tx(
		c,
		evm,
		c.caller,
		destination,
		sendHash.Big(),
		leafNum,
		evm.Context.BlockNumber,
		bigL1BlockNum,
		&blockTime,
		value,
		calldataForL1,
	)

	if c.State.ArbOSVersion() >= 4 {
		return leafNum, nil
	}
	return sendHash.Big(), err
}

// SendMerkleTreeState gets the root, size, and partials of the outbox Merkle tree state (caller must be the 0 address)
func (con ArbSys) SendMerkleTreeState(c ctx, evm mech) (huge, bytes32, []bytes32, error) {
	if c.caller != (addr{}) {
		return nil, bytes32{}, nil, errors.New("method can only be called by address zero")
	}

	// OK to not charge gas, because method is only callable by address zero

	size, rootHash, rawPartials, _ := c.State.SendMerkleAccumulator().StateForExport()
	partials := make([]bytes32, len(rawPartials))
	for i, par := range rawPartials {
		partials[i] = par
	}
	return big.NewInt(int64(size)), rootHash, partials, nil
}

// WithdrawEth send paid eth to the destination on L1
func (con ArbSys) WithdrawEth(c ctx, evm mech, value huge, destination addr) (huge, error) {
	return con.SendTxToL1(c, evm, value, destination, []byte{})
}

func (con ArbSys) isTopLevel(c ctx, evm mech) bool {
	depth := evm.Depth()
	return depth < 2 || evm.Origin == c.txProcessor.Callers[depth-2]
}
