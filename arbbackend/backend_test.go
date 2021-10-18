package arbbackend

import (
	"crypto/ecdsa"
	"io/ioutil"
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/arbstate/arbos"
)

func CreateTestBackendWithBalance(t *testing.T) (*ArbBackend, *ecdsa.PrivateKey) {
	stackConf := node.DefaultConfig
	var err error
	stackConf.DataDir, err = ioutil.TempDir("/tmp", "nitro-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(stackConf.DataDir)
	stackConf.HTTPHost = "localhost"
	stackConf.HTTPModules = append(stackConf.HTTPModules, "eth")
	stack, err := node.New(&stackConf)
	if err != nil {
		if err != nil {
			utils.Fatalf("Error creating protocol stack: %v\n", err)
		}
	}
	nodeConf := ethconfig.Defaults
	nodeConf.NetworkId = arbos.ChainConfig.ChainID.Uint64()

	ownerKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	ownerAddress := crypto.PubkeyToAddress(ownerKey.PublicKey)

	genesisAlloc := make(map[common.Address]core.GenesisAccount)
	genesisAlloc[ownerAddress] = core.GenesisAccount{
		Balance:    big.NewInt(params.Ether),
		Nonce:      0,
		PrivateKey: nil,
	}
	nodeConf.Genesis = &core.Genesis{
		Config:     arbos.ChainConfig,
		Nonce:      0,
		Timestamp:  1633932474,
		ExtraData:  []byte("ArbitrumMainnet"),
		GasLimit:   0,
		Difficulty: big.NewInt(1),
		Mixhash:    common.Hash{},
		Coinbase:   common.Address{},
		Alloc:      genesisAlloc,
		Number:     0,
		GasUsed:    0,
		ParentHash: common.Hash{},
		BaseFee:    big.NewInt(0),
	}

	backend, err := New(stack, &nodeConf)
	if err != nil {
		t.Fatal(err)
	}
	return backend, ownerKey
}

func TestInitial(t *testing.T) {
	backend, ownerKey := CreateTestBackendWithBalance(t)

	signer := types.NewLondonSigner(arbos.ChainConfig.ChainID)
	user2Key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	user2Address := crypto.PubkeyToAddress(user2Key.PublicKey)
	tx, err := types.SignNewTx(ownerKey, signer, &types.DynamicFeeTx{
		ChainID:    arbos.ChainConfig.ChainID,
		Nonce:      0,
		GasTipCap:  &big.Int{},
		GasFeeCap:  &big.Int{},
		Gas:        0,
		To:         &user2Address,
		Value:      big.NewInt(1e10),
		Data:       []byte{},
		AccessList: []types.AccessTuple{},
	})
	if err != nil {
		t.Fatal(err)
	}

	backend.EnqueueL2Message(tx)
}
