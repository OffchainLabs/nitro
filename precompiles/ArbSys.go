//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package precompiles

import (
	"errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/arbstate/arbos"
	"github.com/offchainlabs/arbstate/arbos/util"
	"math/big"
)

type ArbSys struct {
	Address                  addr
	L2ToL1Transaction        func(mech, addr, addr, huge, huge, huge, huge, huge, huge, huge, []byte)
	L2ToL1TransactionGasCost func(addr, addr, huge, huge, huge, huge, huge, huge, huge, []byte) uint64
	SendMerkleUpdate         func(mech, huge, huge, [32]byte)
	SendMerkleUpdateGasCost  func(huge, huge, [32]byte) uint64
}

func (con *ArbSys) ArbBlockNumber(c ctx, evm mech) (huge, error) {
	return evm.Context.BlockNumber, nil
}

func (con *ArbSys) ArbChainID(c ctx, evm mech) (huge, error) {
	return evm.ChainConfig().ChainID, nil
}

func (con *ArbSys) ArbOSVersion(c ctx) (huge, error) {
	return big.NewInt(1000), nil
}

func (con *ArbSys) GetStorageAt(c ctx, evm mech, address addr, index huge) (huge, error) {
	if err := c.burn(params.SloadGas); err != nil {
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
	cost := params.CallValueTransferGas
	zero := new(big.Int)
	dest := destination
	cost += con.SendMerkleUpdateGasCost(zero, zero, common.Hash{})
	cost += con.L2ToL1TransactionGasCost(dest, dest, zero, zero, zero, zero, zero, zero, zero, calldataForL1)
	if err := c.burn(cost); err != nil {
		return nil, err
	}

	sendHash := crypto.Keccak256Hash(common.BigToHash(value).Bytes(), destination.Bytes(), calldataForL1)
	arbosState := arbos.OpenArbosState(evm.StateDB)
	merkleAcc := arbosState.SendMerkleAccumulator()
	merkleUpdateEvent := merkleAcc.Append(sendHash)

	// burn the callvalue, which was previously deposited to this precompile's account
	evm.StateDB.SubBalance(con.Address, value)

	con.SendMerkleUpdate(
		evm,
		big.NewInt(int64(merkleUpdateEvent.Level)),
		big.NewInt(int64(merkleUpdateEvent.LeafNum)),
		merkleUpdateEvent.Hash,
	)

	con.L2ToL1Transaction(
		evm,
		c.caller,
		destination,
		sendHash.Big(),
		big.NewInt(int64(merkleAcc.Size()-1)),
		big.NewInt(0),
		evm.Context.BlockNumber,
		evm.Context.BlockNumber, // TODO: should use Ethereum block number here; currently using Arb block number
		big.NewInt(int64(arbosState.LastTimestampSeen())),
		value,
		calldataForL1,
	)

	return sendHash.Big(), nil
}

func (con ArbSys) SendMerkleTreeState(c ctx, evm mech) (*big.Int, [32]byte, [][32]byte, error) {
	if c.caller != (common.Address{}) {
		return nil, [32]byte{}, nil, errors.New("method can only be called by address zero")
	}

	// OK to not charge gas, because method is only callable by address zero

	size, rootHash, rawPartials := arbos.OpenArbosState(evm.StateDB).SendMerkleAccumulator().StateForExport()
	partials := make([][32]byte, len(rawPartials))
	for i, par := range rawPartials {
		partials[i] = [32]byte(par)
	}
	return big.NewInt(int64(size)), [32]byte(rootHash), partials, nil
}

func (con *ArbSys) WasMyCallersAddressAliased(c ctx, evm mech) (bool, error) {
	return evm.Depth()==2 && evm.OriginWasRemapped, nil
}

func (con ArbSys) WithdrawEth(c ctx, evm mech, value *big.Int, destination common.Address) (*big.Int, error) {
	return con.SendTxToL1(c, evm, value, destination, []byte{})
}
