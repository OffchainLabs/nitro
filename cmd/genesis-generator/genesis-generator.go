package main

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"strings"

	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/cmd/util/confighelpers"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/statetransfer"
	"github.com/offchainlabs/nitro/validator"
)

func main() {
	if err := mainImpl(); err != nil {
		log.Error("Error running genesis generator", "error", err)
		os.Exit(1)
	}
}

func mainImpl() error {
	args := os.Args[1:]
	f := pflag.NewFlagSet("", pflag.ContinueOnError)

	ConfigAddOptions(f)

	k, err := confighelpers.BeginCommonParse(f, args)
	if err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}

	err = confighelpers.ApplyOverrides(f, k)
	if err != nil {
		return fmt.Errorf("failed to apply overrides: %w", err)
	}

	var config Config
	if err := confighelpers.EndCommonParse(k, &config); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	if config.GenesisJsonFile == "" {
		return fmt.Errorf("genesis JSON file must be specified")
	}
	genesisJson, err := os.ReadFile(config.GenesisJsonFile)
	if err != nil {
		return fmt.Errorf("failed to read genesis JSON file %s: %w", config.GenesisJsonFile, err)
	}
	serializedChainConfig, err := extractSerializedChainConfigFromJSON(genesisJson)
	if err != nil {
		return fmt.Errorf("failed to extract serialized chain config from genesis JSON: %w", err)
	}
	var gen core.Genesis
	if err := json.Unmarshal(genesisJson, &gen); err != nil {
		return fmt.Errorf("failed to unmarshal genesis JSON: %w", err)
	}
	var accounts []statetransfer.AccountInitializationInfo
	for address, account := range gen.Alloc {
		accounts = append(accounts, statetransfer.AccountInitializationInfo{
			Addr:       address,
			EthBalance: account.Balance,
			Nonce:      account.Nonce,
			ContractInfo: &statetransfer.AccountInitContractInfo{
				Code:            account.Code,
				ContractStorage: account.Storage,
			},
		})
	}
	initDataReader := statetransfer.NewMemoryInitDataReader(&statetransfer.ArbosInitializationInfo{
		Accounts: accounts,
	})
	chainConfig := gen.Config
	genesisArbOSInit := gen.ArbOSInit
	parsedInitMessage := &arbostypes.ParsedInitMessage{
		ChainId:               chainConfig.ChainID,
		InitialL1BaseFee:      big.NewInt(config.InitialL1BaseFee),
		ChainConfig:           chainConfig,
		SerializedChainConfig: serializedChainConfig,
	}
	genesisBlock, err := generateGenesisBlock(rawdb.NewMemoryDatabase(),
		gethexec.DefaultCacheConfigFor(&config.Caching),
		initDataReader,
		chainConfig,
		genesisArbOSInit,
		parsedInitMessage,
		config.AccountsPerSync,
	)
	if err != nil {
		return fmt.Errorf("failed to generate genesis hash: %w", err)
	}
	// To get send root from genesis block, we need to deserialize the header extra information
	gensisBlockHeader := genesisBlock.Header()
	gensisBlockHeaderInfo := types.DeserializeHeaderExtraInformation(gensisBlockHeader)
	globalState := validator.GoGlobalState{
		BlockHash:  genesisBlock.Hash(),
		SendRoot:   gensisBlockHeaderInfo.SendRoot,
		Batch:      1,
		PosInBatch: 0,
	}
	fmt.Print(globalState)
	return nil
}

func generateGenesisBlock(chainDb ethdb.Database, cacheConfig *core.BlockChainConfig, initData statetransfer.InitDataReader, chainConfig *params.ChainConfig, genesisArbOSInit *params.ArbOSInit, initMessage *arbostypes.ParsedInitMessage, accountsPerSync uint) (*types.Block, error) {
	EmptyHash := common.Hash{}
	prevHash := EmptyHash
	blockNumber, err := initData.GetNextBlockNumber()
	if err != nil {
		return nil, err
	}
	timestamp := uint64(0)
	if blockNumber > 0 {
		prevHash = rawdb.ReadCanonicalHash(chainDb, blockNumber-1)
		if prevHash == EmptyHash {
			return nil, fmt.Errorf("block number %d not found in database", chainDb)
		}
		prevHeader := rawdb.ReadHeader(chainDb, prevHash, blockNumber-1)
		if prevHeader == nil {
			return nil, fmt.Errorf("block header for block %d not found in database", chainDb)
		}
		timestamp = prevHeader.Time
	}
	stateRoot, err := arbosState.InitializeArbosInDatabase(chainDb, cacheConfig, initData, chainConfig, genesisArbOSInit, initMessage, timestamp, accountsPerSync)
	if err != nil {
		return nil, err
	}

	return arbosState.MakeGenesisBlock(prevHash, blockNumber, timestamp, stateRoot, chainConfig), nil
}

func extractSerializedChainConfigFromJSON(genesisJson []byte) ([]byte, error) {
	jsonStr := string(genesisJson)
	// Decode with json.NewDecoder
	decoder := json.NewDecoder(strings.NewReader(jsonStr))

	// Set decoded json feilds to map
	var result map[string]json.RawMessage
	if err := decoder.Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode JSON: %w", err)
	}

	serializedChainConfig, exists := result["config"]
	if !exists {
		return nil, fmt.Errorf("config field not found")
	}

	return serializedChainConfig, nil
}

type Config struct {
	Caching          gethexec.CachingConfig `koanf:"caching"`
	GenesisJsonFile  string                 `koanf:"genesis-json-file"`
	AccountsPerSync  uint                   `koanf:"accounts-per-sync"`
	InitialL1BaseFee int64                  `koanf:"initial-l1-base-fee"`
}

var ConfigDefault = Config{
	Caching:          gethexec.DefaultCachingConfig,
	GenesisJsonFile:  "",
	AccountsPerSync:  100000,
	InitialL1BaseFee: arbostypes.DefaultInitialL1BaseFee.Int64(),
}

func ConfigAddOptions(f *pflag.FlagSet) {
	gethexec.CachingConfigAddOptions("caching", f)
	f.String("genesis-json-file", ConfigDefault.GenesisJsonFile, "path for genesis json file")
	f.Uint("accounts-per-sync", ConfigDefault.AccountsPerSync, "during init - sync database every X accounts. Lower value for low-memory systems. 0 disables.")
	f.Int64("initial-l1-base-fee", ConfigDefault.InitialL1BaseFee, "initial L1 base fee for genesis block")
}
