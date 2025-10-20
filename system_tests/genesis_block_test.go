// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"encoding/json"
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/statetransfer"
)

const chainId = 12345
const chainOwner = "0x0000000000000000000000000000000000000000"

func TestChainConfigSerializationOrderDoesNotChangeGenesisBlockHash(t *testing.T) {
	serializedChainConfig1 := getChainConfig1(chainId)
	serializedChainConfig2 := getChainConfig2(chainId)
	validateSerializedConfigs(t, serializedChainConfig1, serializedChainConfig2)

	genesisHash1 := computeGenesisBlockHash(t, serializedChainConfig1)
	genesisHash2 := computeGenesisBlockHash(t, serializedChainConfig2)

	require.Equal(t, genesisHash1, genesisHash2, "Genesis block hashes should be equal despite different serialization orders")
}

func validateSerializedConfigs(t *testing.T, serializedConfig1, serializedConfig2 []byte) {
	// Sanity check to ensure that the two serializations are different
	require.NotEqual(t, serializedConfig1, serializedConfig2)
	// Sanity check to ensure that the two serializations encode the same object
	var deserializedChainConfig1, deserializedChainConfig2 params.ChainConfig
	err := json.Unmarshal(serializedConfig1, &deserializedChainConfig1)
	require.NoError(t, err)
	err = json.Unmarshal(serializedConfig2, &deserializedChainConfig2)
	require.NoError(t, err)
	require.Equal(t, deserializedChainConfig1, deserializedChainConfig2)
}

func getChainConfig1(chainId uint64) []byte {
	return []byte(fmt.Sprintf(chainConfig1, chainId, chainOwner))
}

func getChainConfig2(chainId uint64) []byte {
	return []byte(fmt.Sprintf(chainConfig2, chainId, chainOwner))
}

func computeGenesisBlockHash(t *testing.T, serializedChainConfig []byte) common.Hash {
	var chainConfig params.ChainConfig
	err := json.Unmarshal(serializedChainConfig, &chainConfig)
	require.NoError(t, err)

	parsedInitMessage := &arbostypes.ParsedInitMessage{
		ChainId:          chainConfig.ChainID,
		InitialL1BaseFee: big.NewInt(1000000000),
		ChainConfig:      &chainConfig,
	}

	genesisBlock := generateGenesisBlock(
		t,
		&chainConfig,
		parsedInitMessage,
	)
	require.NoError(t, err)

	return genesisBlock.Hash()
}

// Taken from cmd/genesis-generator/genesis-generator.go and adapted slightly
func generateGenesisBlock(
	t *testing.T,
	chainConfig *params.ChainConfig,
	initMessage *arbostypes.ParsedInitMessage,
) *types.Block {
	timestamp := uint64(0)
	stateRoot, err := arbosState.InitializeArbosInDatabase(
		rawdb.NewMemoryDatabase(),
		gethexec.DefaultCacheConfigFor(&gethexec.DefaultCachingConfig),
		statetransfer.NewMemoryInitDataReader(&statetransfer.ArbosInitializationInfo{
			ChainOwner: common.HexToAddress(chainOwner),
		}),
		chainConfig,
		&params.ArbOSInit{},
		initMessage,
		timestamp,
		10,
	)
	require.NoError(t, err)

	return arbosState.MakeGenesisBlock(common.Hash{}, 0, timestamp, stateRoot, chainConfig)
}

const chainConfig1 = `{
  "chainId": %d,
  "daoForkSupport": true,
  "depositContractAddress": "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
  "enableVerkleAtGenesis": false,
  "ethash": {},
  "clique": {
	"period": 1,
	"epoch": 1
  },
  "arbitrum": {	
	"EnableArbOS": true,
	"AllowDebugPrecompiles": true,
	"DataAvailabilityCommittee": false,
	"InitialArbOSVersion": 50,
	"InitialChainOwner": "%s",
	"GenesisBlockNum": 0,
	"MaxCodeSize": 0,
	"MaxInitCodeSize": 0
  }
}`

// Changed order of fields:
//   - daoForkSupport is now first, chainId is second
//   - clique fields order changed
const chainConfig2 = `{
  "daoForkSupport": true,
  "chainId": %d,
  "depositContractAddress": "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
  "enableVerkleAtGenesis": false,
  "ethash": {},
  "clique": {
	"epoch": 1,
	"period": 1
  },
  "arbitrum": {	
	"EnableArbOS": true,
	"AllowDebugPrecompiles": true,
	"DataAvailabilityCommittee": false,
	"InitialArbOSVersion": 50,
	"InitialChainOwner": "%s",
	"GenesisBlockNum": 0,
	"MaxCodeSize": 0,
	"MaxInitCodeSize": 0
  }
}`
