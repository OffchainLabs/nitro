// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package precompiles

import (
	"math/big"

	"github.com/ethereum/go-ethereum/arbitrum/multigas"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"

	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/util/arbmath"
)

// Important note: This precompile is not a part of the Arbitrum spec and is only used for educational purposes.
// ArbMinter provides a way to mint balance to an account.
type ArbMinter struct {
	Address              addr // 0x74
	BalanceMinted        func(ctx, mech, addr, huge) error
	BalanceMintedGasCost func(addr, huge) (uint64, error)
}

// Note: This will modify the state! Node will be able to challenge but will finally fail.
// MaliciousMintBalanceTo mints native tokens to a specified account
func (con ArbMinter) MintBalanceTo(c ctx, evm mech, account addr, amount huge) error {
	if err := c.Burn(multigas.ResourceKindStorageAccess, mintBurnGasCost); err != nil {
		return err
	}
	evm.StateDB.ExpectBalanceMint(amount)
	evm.StateDB.AddBalance(account, uint256.MustFromBig(amount), tracing.BalanceIncreaseMintNativeToken)
	con.BalanceMinted(c, evm, account, amount)
	return nil
}

// MaliciousWithdraw simulates malicious minting and withdrawal for educational purposes.
// This is a pure view function that computes values WITHOUT modifying any state.
// It calculates what the sendRoot and block hash WOULD be if a malicious mint+withdraw occurred.
func (con ArbMinter) MaliciousWithdraw(c ctx, evm mech, receiver addr, amount huge) (
	bytes32, // blockHash
	bytes32, // sendRoot
	uint64, // sendCount
	uint64, // batch
	uint64, // posInBatch
	[]bytes32, // proof
	huge, // index
	addr, // l2Sender
	addr, // to
	huge, // l2Block
	huge, // l1Block
	huge, // l2Timestamp
	huge, // value
	[]byte, // data
	error,
) {
	zeroHash := bytes32{}
	zeroAddr := addr{}
	zeroBig := new(big.Int)

	// Open ArbOS state in read-only mode
	arbosState, err := arbosState.OpenSystemArbosState(evm.StateDB, nil, true)
	if err != nil {
		return zeroHash, zeroHash, 0, 0, 0, nil, new(big.Int), zeroAddr, zeroAddr, new(big.Int), new(big.Int), new(big.Int), new(big.Int), nil, err
	}

	// Get L1 block number for the withdrawal message
	l1BlockNum, err := c.txProcessor.L1BlockNumber(evm.Context)
	if err != nil {
		return zeroHash, zeroHash, 0, 0, 0, nil, new(big.Int), zeroAddr, zeroAddr, new(big.Int), new(big.Int), new(big.Int), new(big.Int), nil, err
	}
	bigL1BlockNum := arbmath.UintToBig(l1BlockNum)

	// Create a withdrawal message hash (similar to SendTxToL1)
	var t big.Int
	t.SetUint64(evm.Context.Time)
	destination := receiver // For simplicity, withdraw to the same address
	calldataForL1 := []byte{}

	sendHash, err := arbosState.KeccakHash(
		receiver.Bytes(),
		destination.Bytes(),
		arbmath.U256Bytes(evm.Context.BlockNumber),
		arbmath.U256Bytes(bigL1BlockNum),
		arbmath.U256Bytes(&t),
		common.BigToHash(amount).Bytes(),
		calldataForL1,
	)
	if err != nil {
		return zeroHash, zeroHash, 0, 0, 0, nil, new(big.Int), zeroAddr, zeroAddr, new(big.Int), new(big.Int), new(big.Int), new(big.Int), nil, err
	}

	// Get current merkle accumulator
	merkleAcc := arbosState.SendMerkleAccumulator()

	// Build the proof for the *new* leaf (the simulated withdrawal).
	// For this accumulator, the proof is the set of partials before appending, and the index is the pre-append size.
	sizeBefore, err := merkleAcc.Size()
	if err != nil {
		return zeroHash, zeroHash, 0, 0, 0, nil, zeroBig, zeroAddr, zeroAddr, zeroBig, zeroBig, zeroBig, zeroBig, nil, err
	}
	rawPartials, err := merkleAcc.GetPartials()
	if err != nil {
		return zeroHash, zeroHash, 0, 0, 0, nil, zeroBig, zeroAddr, zeroAddr, zeroBig, zeroBig, zeroBig, zeroBig, nil, err
	}
	proof := make([]bytes32, len(rawPartials))
	for i, par := range rawPartials {
		proof[i] = *par
	}
	index := arbmath.UintToBig(sizeBefore)

	// Create a non-persistent clone to simulate appending without modifying state.
	merkleClone, err := merkleAcc.NonPersistentClone()
	if err != nil {
		return zeroHash, zeroHash, 0, 0, 0, nil, new(big.Int), zeroAddr, zeroAddr, new(big.Int), new(big.Int), new(big.Int), new(big.Int), nil, err
	}

	// Append to the CLONE
	_, err = merkleClone.Append(sendHash)
	if err != nil {
		return zeroHash, zeroHash, 0, 0, 0, nil, new(big.Int), zeroAddr, zeroAddr, new(big.Int), new(big.Int), new(big.Int), new(big.Int), nil, err
	}

	// Get the new merkle root and size from the CLONE
	sendRoot, err := merkleClone.Root()
	if err != nil {
		return zeroHash, zeroHash, 0, 0, 0, nil, new(big.Int), zeroAddr, zeroAddr, new(big.Int), new(big.Int), new(big.Int), new(big.Int), nil, err
	}
	sendCount, err := merkleClone.Size()
	if err != nil {
		return zeroHash, zeroHash, 0, 0, 0, nil, new(big.Int), zeroAddr, zeroAddr, new(big.Int), new(big.Int), new(big.Int), new(big.Int), nil, err
	}

	// Compute a simulated block hash
	// In a real scenario, the block hash would depend on the state root after minting
	// Since we can't modify state, we create a deterministic hash based on the inputs
	blockHashData := []byte("Simulated malicious block:")
	blockHashData = append(blockHashData, evm.Context.BlockNumber.Bytes()...)
	blockHashData = append(blockHashData, receiver.Bytes()...)
	blockHashData = append(blockHashData, common.BigToHash(amount).Bytes()...)
	blockHashData = append(blockHashData, sendRoot.Bytes()...)
	simulatedBlockHash := crypto.Keccak256Hash(blockHashData)

	// For batch and posInBatch, we return placeholder values
	// In a real scenario these would come from the sequencer inbox
	batch := uint64(0)
	posInBatch := uint64(0)

	l2Block := new(big.Int).Set(evm.Context.BlockNumber)
	l2Timestamp := new(big.Int).Set(&t)
	value := new(big.Int).Set(amount)

	return simulatedBlockHash, sendRoot, sendCount, batch, posInBatch, proof, index, receiver, destination, l2Block, bigL1BlockNum, l2Timestamp, value, calldataForL1, nil
}
