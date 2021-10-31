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
	SendMerkleUpdate         func(mech, huge, [32]byte)
	SendMerkleUpdateGasCost  func(huge, [32]byte) uint64
}

func (con *ArbSys) ArbBlockNumber(c ctx, evm mech) (huge, error) {
	return nil, errors.New("unimplemented")
}

func (con *ArbSys) ArbChainID(c ctx, evm mech) (huge, error) {
	return big.NewInt(412345), nil
}

func (con *ArbSys) ArbOSVersion(c ctx) (huge, error) {
	return big.NewInt(1000), nil
}

func (con *ArbSys) GetStorageAt(c ctx, evm mech, address addr, index huge) (huge, error) {
	return nil, errors.New("unimplemented")
}

func (con *ArbSys) GetStorageGasAvailable(c ctx, evm mech) (huge, error) {
	return nil, errors.New("unimplemented")
}

func (con *ArbSys) GetTransactionCount(c ctx, evm mech, account addr) (huge, error) {
	return nil, errors.New("unimplemented")
}

func (con *ArbSys) IsTopLevelCall(c ctx, evm mech) (bool, error) {
	return false, errors.New("unimplemented")
}

func (con *ArbSys) MapL1SenderContractAddressToL2Alias(c ctx, sender addr, dest addr) (addr, error) {
	return addr{}, errors.New("unimplemented")
}

func (con *ArbSys) MyCallersAddressWithoutAliasing(c ctx, evm mech) (addr, error) {
	return addr{}, errors.New("unimplemented")
}

func (con *ArbSys) SendTxToL1(c ctx, evm mech, value huge, destination addr, calldataForL1 []byte) (*big.Int, error) {

	cost := params.CallValueTransferGas
	zero := new(big.Int)
	dest := destination
	cost += 2 * con.SendMerkleUpdateGasCost(zero, common.Hash{})
	cost += con.L2ToL1TransactionGasCost(dest, dest, zero, zero, zero, zero, zero, zero, zero, calldataForL1)
	if err := c.burn(cost); err != nil {
		return nil, err
	}

	sendHash := crypto.Keccak256Hash(common.BigToHash(value).Bytes(), destination.Bytes(), calldataForL1)
	arbosState := arbos.OpenArbosState(evm.StateDB)
	merkleAcc := arbosState.SendMerkleAccumulator()
	merkleUpdateEvents := merkleAcc.Append(sendHash)

	// burn the callvalue, which was previously deposited to this precompile's account
	evm.StateDB.SubBalance(con.Address, value)

	for _, merkleUpdateEvent := range merkleUpdateEvents {
		levelAndLeafNum := new(big.Int).Add(
			new(big.Int).Lsh(big.NewInt(1), uint(merkleUpdateEvent.Level)),
			big.NewInt(int64(merkleUpdateEvent.NumLeaves)),
		)
		con.SendMerkleUpdate(
			evm,
			levelAndLeafNum,
			merkleUpdateEvent.Hash,
		)
	}

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
	return false, errors.New("unimplemented")
}

func (con ArbSys) WithdrawEth(c ctx, evm mech, value *big.Int, destination common.Address) (*big.Int, error) {
	return con.SendTxToL1(c, evm, value, destination, []byte{})
}
