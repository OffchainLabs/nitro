package assertionchain

import (
	"crypto/ecdsa"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"math/big"
	"testing"
)

func TestDoSomething(t *testing.T) {
	if 1 == 1 {
		t.Error("failed")
	}
}

// Represents a test EOA account in the simulated backend,
type testAccount struct {
	accountAddr common.Address
	Backend     *backends.SimulatedBackend
	TxOpts      *bind.TransactOpts
}

func setupAccount() {
	_ = backends.SimulatedBackend{}
}

func setup() (*testAccount, error) {
	genesis := make(core.GenesisAlloc)
	privKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}
	pubKeyECDSA, ok := privKey.Public().(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("error casting public key to ECDSA")
	}

	// Strip off the 0x and the first 2 characters 04 which is always the
	// EC prefix and is not required.
	publicKeyBytes := crypto.FromECDSAPub(pubKeyECDSA)[4:]
	var pubKey = make([]byte, 48)
	copy(pubKey, publicKeyBytes)

	addr := crypto.PubkeyToAddress(privKey.PublicKey)
	chainID := big.NewInt(1337)
	txOpts, err := bind.NewKeyedTransactorWithChainID(privKey, chainID)
	if err != nil {
		return nil, err
	}
	startingBalance, _ := new(big.Int).SetString(
		"100000000000000000000000000000000000000",
		10,
	)
	genesis[addr] = core.GenesisAccount{Balance: startingBalance}
	gasLimit := uint64(2100000000000)
	backend := backends.NewSimulatedBackend(genesis, gasLimit)

	// contractAddr, _, contract, err := DeployDepositContract(txOpts, backend)
	// if err != nil {
	// 	return nil, err
	// }
	// backend.Commit()

	return &testAccount{
		accountAddr: addr,
		Backend:     backend,
		TxOpts:      txOpts,
	}, nil
}
