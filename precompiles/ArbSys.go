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
	"math/big"
)

type ArbSys struct {
	Address                  addr
	L2ToL1Transaction        func(mech, addr, addr, huge, huge, huge, huge, huge, huge, huge, []byte)
	L2ToL1TransactionGasCost func(addr, addr, huge, huge, huge, huge, huge, huge, huge, []byte) uint64
	SendMerkleUpdate         func(mech, huge, huge, [32]byte)
	SendMerkleUpdateGasCost  func(huge, huge, [32]byte) uint64
}

func (con *ArbSys) ArbBlockNumber(caller addr, evm mech) (huge, error) {
	return nil, errors.New("unimplemented")
}

func (con *ArbSys) ArbBlockNumberGasCost() uint64 {
	return 0
}

func (con *ArbSys) ArbChainID(caller addr, evm mech) (huge, error) {
	return big.NewInt(412345), nil
}

func (con *ArbSys) ArbChainIDGasCost() uint64 {
	return 0
}

func (con *ArbSys) ArbOSVersion(caller addr) (huge, error) {
	return big.NewInt(1000), nil
}

func (con *ArbSys) ArbOSVersionGasCost() uint64 {
	return 0
}

func (con *ArbSys) GetStorageAt(caller addr, evm mech, address addr, index huge) (huge, error) {
	return nil, errors.New("unimplemented")
}

func (con *ArbSys) GetStorageAtGasCost(address addr, index huge) uint64 {
	return 0
}

func (con *ArbSys) GetStorageGasAvailable(caller addr, evm mech) (huge, error) {
	return nil, errors.New("unimplemented")
}

func (con *ArbSys) GetStorageGasAvailableGasCost() uint64 {
	return 0
}

func (con *ArbSys) GetTransactionCount(caller addr, evm mech, account addr) (huge, error) {
	return nil, errors.New("unimplemented")
}

func (con *ArbSys) GetTransactionCountGasCost(account addr) uint64 {
	return 0
}

func (con *ArbSys) IsTopLevelCall(caller addr, evm mech) (bool, error) {
	return false, errors.New("unimplemented")
}

func (con *ArbSys) IsTopLevelCallGasCost() uint64 {
	return 0
}

func (con *ArbSys) MapL1SenderContractAddressToL2Alias(
	caller addr,
	sender addr,
	dest addr,
) (addr, error) {
	return addr{}, errors.New("unimplemented")
}

func (con *ArbSys) MapL1SenderContractAddressToL2AliasGasCost(sender addr, dest addr) uint64 {
	return 0
}

func (con *ArbSys) MyCallersAddressWithoutAliasing(caller addr, evm mech) (addr, error) {
	return addr{}, errors.New("unimplemented")
}

func (con *ArbSys) MyCallersAddressWithoutAliasingGasCost() uint64 {
	return 0
}

func (con *ArbSys) SendTxToL1(
	caller addr,
	evm mech,
	value huge,
	destination addr,
	calldataForL1 []byte,
) (*big.Int, error) {
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
		caller,
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

func (con ArbSys) SendTxToL1GasCost(destination common.Address, calldataForL1 []byte) uint64 {
	return params.CallValueTransferGas +
		2*params.LogGas +
		6*params.LogTopicGas +
		(10*32+uint64(len(calldataForL1)))*params.LogDataGas
}

func (con ArbSys) SendMerkleTreeState(caller addr, evm mech) (*big.Int, [32]byte, [][32]byte, error) {
	if caller != (common.Address{}) {
		return nil, [32]byte{}, nil, errors.New("method can only be called by address zero")
	}
	size, rootHash, rawPartials := arbos.OpenArbosState(evm.StateDB).SendMerkleAccumulator().StateForExport()
	partials := make([][32]byte, len(rawPartials))
	for i, par := range rawPartials {
		partials[i] = [32]byte(par)
	}
	return big.NewInt(int64(size)), [32]byte(rootHash), partials, nil
}

func (con ArbSys) SendMerkleTreeStateGasCost() uint64 {
	return 0 // OK to leave it at zero, because method is only callable by address zero
}

func (con *ArbSys) WasMyCallersAddressAliased(caller addr, evm mech) (bool, error) {
	return false, errors.New("unimplemented")
}

func (con *ArbSys) WasMyCallersAddressAliasedGasCost() uint64 {
	return 0
}

func (con ArbSys) WithdrawEth(
	caller common.Address,
	evm mech,
	value *big.Int,
	destination common.Address,
) (*big.Int, error) {
	return con.SendTxToL1(caller, evm, value, destination, []byte{})
}

func (con ArbSys) WithdrawEthGasCost(destination common.Address) uint64 {
	return con.SendTxToL1GasCost(destination, []byte{})
}
