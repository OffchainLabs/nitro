package main

import (
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/offchainlabs/arbio"
)

func main() {
	// TODO host API data retrieval
	rawdb := rawdb.NewMemoryDatabase()
	db := state.NewDatabase(rawdb)
	blockHashRetriever := func(height uint64) common.Hash {
		if height > 0 {
			panic(fmt.Sprintf("Attempted to retrieve hash of unknown block height %v", height))
		}
		return common.Hash{}
	}
	key, err := crypto.HexToECDSA("7f9db344131e31e74219902df2911944d7ef40c5b6d6a7f2c51e224f98637468")
	if err != nil {
		panic(fmt.Sprintf("Error parsing ECDSA key: %v", err))
	}
	signer, err := bind.NewKeyedTransactorWithChainID(key, arbio.CHAIN_ID)
	if err != nil {
		panic(fmt.Sprintf("Error creating signer: %v", err))
	}
	var data []byte
	// Write the int 2 to the storage slot 1
	data = append(data, 0x60, 0x02, 0x60, 0x01, 0x55)
	// Write contract code with one invalid instruction to memory
	data = append(data, 0x60, 0xFE, 0x60, 0x00, 0x53)
	// Return the contract code
	data = append(data, 0x60, 0x01, 0x60, 0x00, 0xF3)
	tx := types.NewTx(&types.LegacyTx{
		Nonce:    0,
		GasPrice: big.NewInt(1),
		Gas:      1_000_000,
		To:       nil,
		Value:    big.NewInt(1e9),
		Data:     data,
	})
	tx, err = signer.Signer(signer.From, tx)
	if err != nil {
		panic(fmt.Sprintf("Error signing transaction: %v", err))
	}
	msg := arbio.ArbMessage{
		From:    signer.From,
		Deposit: big.NewInt(1e18),
		Tx:      tx,
	}
	var lastStateRoot common.Hash

	fmt.Printf("Previous state root: %v\n", lastStateRoot)
	newStateRoot, err := arbio.Process(db, blockHashRetriever, lastStateRoot, 1, big.NewInt(1), msg)
	if err != nil {
		fmt.Printf("Error processing message: %v\n", err)
		newStateRoot = lastStateRoot
	}
	fmt.Printf("New state root: %v\n", newStateRoot)

	statedb, err := state.New(newStateRoot, db, nil)
	if err == nil {
		fmt.Printf("Sender address: %v\n", signer.From.String())
		contractAddr := crypto.CreateAddress(signer.From, 0)
		fmt.Printf("Contract address: %v\n", contractAddr.String())
		senderBalance := statedb.GetBalance(signer.From)
		fmt.Printf("Sender balance: %v\n", senderBalance.String())
		contractBalance := statedb.GetBalance(contractAddr)
		fmt.Printf("Contract balance: %v\n", contractBalance.String())
		storageKey := [32]byte{}
		storageKey[31] = 1
		code := statedb.GetCode(contractAddr)
		fmt.Printf("Contract code: 0x%v\n", hex.EncodeToString(code))
		value := statedb.GetState(contractAddr, storageKey)
		fmt.Printf("Storage slot value: %v\n", value.String())
	} else {
		fmt.Printf("Error opening new state db: %v\n", err)
	}
}
