// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

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

	chainConfig, serializedChainConfig, err := readChainConfig(&gen)
	if err != nil {
		return err
	}

	genesisArbOSInit := gen.ArbOSInit
	if genesisArbOSInit == nil {
		return errors.New("genesis ArbOS init was not set (`arbOSInit`)")
	}

	parsedInitMessage, err := buildInitMessage(genesisArbOSInit, chainConfig, serializedChainConfig)
	if err != nil {
		return err
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

func generateGenesisBlock(executionDB ethdb.Database, cacheConfig *core.BlockChainConfig, initData statetransfer.InitDataReader, chainConfig *params.ChainConfig, genesisArbOSInit *params.ArbOSInit, initMessage *arbostypes.ParsedInitMessage, accountsPerSync uint) (*types.Block, error) {
	EmptyHash := common.Hash{}
	prevHash := EmptyHash
	blockNumber, err := initData.GetNextBlockNumber()
	if err != nil {
		return nil, err
	}
	timestamp := uint64(0)
	if blockNumber > 0 {
		prevHash = rawdb.ReadCanonicalHash(executionDB, blockNumber-1)
		if prevHash == EmptyHash {
			return nil, fmt.Errorf("block number %d not found in database", executionDB)
		}
		prevHeader := rawdb.ReadHeader(executionDB, prevHash, blockNumber-1)
		if prevHeader == nil {
			return nil, fmt.Errorf("block header for block %d not found in database", executionDB)
		}
		timestamp = prevHeader.Time
	}
	stateRoot, err := arbosState.InitializeArbosInDatabase(executionDB, cacheConfig, initData, chainConfig, genesisArbOSInit, initMessage, timestamp, accountsPerSync)
	if err != nil {
		return nil, err
	}

	return arbosState.MakeGenesisBlock(prevHash, blockNumber, timestamp, stateRoot, chainConfig), nil
}

func readChainConfig(gen *core.Genesis) (*params.ChainConfig, []byte, error) {
	// 1. Validate that the correct fields are used
	if gen.Config != nil {
		return nil, nil, errors.New("`config` field is deprecated and not supported; use `serializedChainConfig` instead")
	}
	if gen.SerializedChainConfig == "" {
		return nil, nil, errors.New("serialized chain config was not set (`serializedChainConfig`)")
	}
	// 2. Deserialize the chain config
	chainConfig, err := gen.GetConfig()
	if err != nil {
		return nil, nil, err
	}
	return chainConfig, []byte(gen.SerializedChainConfig), nil
}

func buildInitMessage(genesisArbOSInit *params.ArbOSInit, chainConfig *params.ChainConfig, serializedChainConfig []byte) (*arbostypes.ParsedInitMessage, error) {
	if genesisArbOSInit.InitialL1BaseFee == nil {
		return nil, errors.New("initial L1 base fee was not set (`arbOSInit.initialL1BaseFee`)")
	}
	if chainConfig.ChainID == nil {
		return nil, fmt.Errorf("chain ID was not set (`serializedChainConfig.chainId`)")
	}

	return &arbostypes.ParsedInitMessage{
		ChainId:               chainConfig.ChainID,
		InitialL1BaseFee:      genesisArbOSInit.InitialL1BaseFee,
		ChainConfig:           chainConfig,
		SerializedChainConfig: serializedChainConfig,
	}, nil
}

type Config struct {
	Caching         gethexec.CachingConfig `koanf:"caching"`
	GenesisJsonFile string                 `koanf:"genesis-json-file"`
	AccountsPerSync uint                   `koanf:"accounts-per-sync"`
}

var ConfigDefault = Config{
	Caching:         gethexec.DefaultCachingConfig,
	GenesisJsonFile: "",
	AccountsPerSync: 100000,
}

func ConfigAddOptions(f *pflag.FlagSet) {
	gethexec.CachingConfigAddOptions("caching", f)
	f.String("genesis-json-file", ConfigDefault.GenesisJsonFile, "path for genesis json file")
	f.Uint("accounts-per-sync", ConfigDefault.AccountsPerSync, "during init - sync database every X accounts. Lower value for low-memory systems. 0 disables.")
}
