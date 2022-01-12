//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package precompiles

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/offchainlabs/arbstate/arbos/storage"
	"github.com/offchainlabs/arbstate/arbos/util"
	"github.com/offchainlabs/arbstate/util/merkletree"
)

type ArbSys struct {
	Address                  addr
	L2ToL1Transaction        func(ctx, mech, addr, addr, huge, huge, huge, huge, huge, huge, huge, []byte) error
	L2ToL1TransactionGasCost func(addr, addr, huge, huge, huge, huge, huge, huge, huge, []byte) (uint64, error)
	SendMerkleUpdate         func(ctx, mech, huge, [32]byte, huge) error
	SendMerkleUpdateGasCost  func(huge, [32]byte, huge) (uint64, error)
}

var InvalidBlockNum = errors.New("Invalid block number")

func (con *ArbSys) ArbBlockNumber(c ctx, evm mech) (huge, error) {
	return evm.Context.BlockNumber, nil
}

func (con *ArbSys) ArbBlockHash(c ctx, evm mech, arbBlockNumber *big.Int) ([32]byte, error) {
	if !arbBlockNumber.IsUint64() {
		return [32]byte{}, InvalidBlockNum
	}
	requestedBlockNum := arbBlockNumber.Uint64()

	var upper, lower uint64
	upper = evm.Context.BlockNumber.Uint64()
	if upper < 257 {
		lower = 0
	} else {
		lower = upper - 256
	}
	if requestedBlockNum < lower || requestedBlockNum >= upper {
		return [32]byte{}, InvalidBlockNum
	}

	return evm.Context.GetHash(requestedBlockNum), nil
}

func (con *ArbSys) ArbChainID(c ctx, evm mech) (huge, error) {
	return evm.ChainConfig().ChainID, nil
}

func (con *ArbSys) ArbOSVersion(c ctx) (huge, error) {
	return big.NewInt(1000), nil
}

func (con *ArbSys) GetStorageAt(c ctx, evm mech, address addr, index huge) (huge, error) {
	if err := c.Burn(storage.StorageReadCost); err != nil {
		return nil, err
	}
	return evm.StateDB.GetState(address, common.BigToHash(index)).Big(), nil
}

func (con *ArbSys) GetStorageGasAvailable(c ctx, evm mech) (huge, error) {
	return big.NewInt(0), nil
}

func (con *ArbSys) GetTransactionCount(c ctx, evm mech, account addr) (huge, error) {
	return big.NewInt(int64(evm.StateDB.GetNonce(account))), nil
}

func (con *ArbSys) IsTopLevelCall(c ctx, evm mech) (bool, error) {
	// need to modify EVM to add a getter for its depth field
	return false, errors.New("unimplemented")
}

func (con *ArbSys) MapL1SenderContractAddressToL2Alias(c ctx, sender addr, dest addr) (addr, error) {
	return util.RemapL1Address(sender), nil
}

func (con *ArbSys) MyCallersAddressWithoutAliasing(c ctx, evm mech) (addr, error) {
	// need special support to enable this
	return addr{}, errors.New("unimplemented")
}

func (con *ArbSys) SendTxToL1(c ctx, evm mech, value huge, destination addr, calldataForL1 []byte) (*big.Int, error) {

	sendHash := crypto.Keccak256Hash(c.caller.Bytes(), common.BigToHash(value).Bytes(), destination.Bytes(), calldataForL1)
	arbosState := c.state
	merkleAcc := arbosState.SendMerkleAccumulator()
	merkleUpdateEvents, err := merkleAcc.Append(sendHash)
	if err != nil {
		return nil, err
	}

	// burn the callvalue, which was previously deposited to this precompile's account
	evm.StateDB.SubBalance(con.Address, value)

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

	size, _ := merkleAcc.Size()
	timestamp, err := arbosState.LastTimestampSeen()
	if err != nil {
		return nil, err
	}

	leafNum := big.NewInt(int64(size - 1))

	err = con.L2ToL1Transaction(
		c,
		evm,
		c.caller,
		destination,
		sendHash.Big(),
		leafNum,
		big.NewInt(0),
		evm.Context.BlockNumber,
		evm.Context.BlockNumber, // TODO: should use Ethereum block number here; currently using Arb block number
		big.NewInt(int64(timestamp)),
		value,
		calldataForL1,
	)

	return sendHash.Big(), err
}

func (con ArbSys) SendMerkleTreeState(c ctx, evm mech) (*big.Int, [32]byte, [][32]byte, error) {
	if c.caller != (common.Address{}) {
		return nil, [32]byte{}, nil, errors.New("method can only be called by address zero")
	}

	// OK to not charge gas, because method is only callable by address zero

	size, rootHash, rawPartials, _ := c.state.SendMerkleAccumulator().StateForExport()
	partials := make([][32]byte, len(rawPartials))
	for i, par := range rawPartials {
		partials[i] = [32]byte(par)
	}
	return big.NewInt(int64(size)), [32]byte(rootHash), partials, nil
}

func (con *ArbSys) WasMyCallersAddressAliased(c ctx, evm mech) (bool, error) {
	return false, errors.New("unimplemented")
}

func (con ArbSys) WithdrawEth(c ctx, evm mech, value *big.Int, destination common.Address) (*big.Int, error) {
	return con.SendTxToL1(c, evm, value, destination, []byte{})
}
