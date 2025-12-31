// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package config

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/offchainlabs/nitro/arbnode"
	blocksreexecutor "github.com/offchainlabs/nitro/blocks_reexecutor"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/cmd/conf"
	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/cmd/util/confighelpers"
	"github.com/offchainlabs/nitro/daprovider/anytrust"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/util/colors"
	"github.com/offchainlabs/nitro/validator/valnode"
	"github.com/spf13/pflag"
)

type NodeConfig struct {
	Conf                   genericconf.ConfConfig          `koanf:"conf" reload:"hot"`
	Node                   arbnode.Config                  `koanf:"node" reload:"hot"`
	Execution              gethexec.Config                 `koanf:"execution" reload:"hot"`
	Validation             valnode.Config                  `koanf:"validation" reload:"hot"`
	ParentChain            conf.ParentChainConfig          `koanf:"parent-chain" reload:"hot"`
	Chain                  conf.L2Config                   `koanf:"chain"`
	LogLevel               string                          `koanf:"log-level" reload:"hot"`
	LogType                string                          `koanf:"log-type" reload:"hot"`
	FileLogging            genericconf.FileLoggingConfig   `koanf:"file-logging" reload:"hot"`
	Persistent             conf.PersistentConfig           `koanf:"persistent"`
	HTTP                   genericconf.HTTPConfig          `koanf:"http"`
	WS                     genericconf.WSConfig            `koanf:"ws"`
	IPC                    genericconf.IPCConfig           `koanf:"ipc"`
	Auth                   genericconf.AuthRPCConfig       `koanf:"auth"`
	GraphQL                genericconf.GraphQLConfig       `koanf:"graphql"`
	Metrics                bool                            `koanf:"metrics"`
	MetricsServer          genericconf.MetricsServerConfig `koanf:"metrics-server"`
	PProf                  bool                            `koanf:"pprof"`
	PprofCfg               genericconf.PProf               `koanf:"pprof-cfg"`
	Init                   conf.InitConfig                 `koanf:"init"`
	Rpc                    genericconf.RpcConfig           `koanf:"rpc"`
	BlocksReExecutor       blocksreexecutor.Config         `koanf:"blocks-reexecutor"`
	EnsureRollupDeployment bool                            `koanf:"ensure-rollup-deployment" reload:"hot"`
}

var NodeConfigDefault = NodeConfig{
	Conf:                   genericconf.ConfConfigDefault,
	Node:                   arbnode.ConfigDefault,
	Execution:              gethexec.ConfigDefault,
	Validation:             valnode.DefaultValidationConfig,
	ParentChain:            conf.L1ConfigDefault,
	Chain:                  conf.L2ConfigDefault,
	LogLevel:               "INFO",
	LogType:                "plaintext",
	FileLogging:            genericconf.DefaultFileLoggingConfig,
	Persistent:             conf.PersistentConfigDefault,
	HTTP:                   genericconf.HTTPConfigDefault,
	WS:                     genericconf.WSConfigDefault,
	IPC:                    genericconf.IPCConfigDefault,
	Auth:                   genericconf.AuthRPCConfigDefault,
	GraphQL:                genericconf.GraphQLConfigDefault,
	Metrics:                false,
	MetricsServer:          genericconf.MetricsServerConfigDefault,
	Init:                   conf.InitConfigDefault,
	Rpc:                    genericconf.DefaultRpcConfig,
	PProf:                  false,
	PprofCfg:               genericconf.PProfDefault,
	BlocksReExecutor:       blocksreexecutor.DefaultConfig,
	EnsureRollupDeployment: true,
}

func (c *NodeConfig) ResolveDirectoryNames() error {
	err := c.Persistent.ResolveDirectoryNames()
	if err != nil {
		return err
	}
	c.Chain.ResolveDirectoryNames(c.Persistent.Chain)

	return nil
}

func (c *NodeConfig) ShallowClone() *NodeConfig {
	config := &NodeConfig{}
	*config = *c
	return config
}

func (c *NodeConfig) CanReload(new *NodeConfig) error {
	var check func(node, other reflect.Value, path string)
	var err error

	check = func(node, value reflect.Value, path string) {
		if node.Kind() != reflect.Struct {
			return
		}

		for i := 0; i < node.NumField(); i++ {
			fieldTy := node.Type().Field(i)
			if !fieldTy.IsExported() {
				continue
			}
			hot := fieldTy.Tag.Get("reload") == "hot"
			dot := path + "." + fieldTy.Name

			first := node.Field(i).Interface()
			other := value.Field(i).Interface()

			if !hot && !reflect.DeepEqual(first, other) {
				err = fmt.Errorf("illegal change to %v%v%v", colors.Red, dot, colors.Clear)
			} else {
				check(node.Field(i), value.Field(i), dot)
			}
		}
	}

	check(reflect.ValueOf(c).Elem(), reflect.ValueOf(new).Elem(), "config")
	return err
}

func (c *NodeConfig) Validate() error {
	if c.Init.RecreateMissingStateFrom > 0 && !c.Execution.Caching.Archive {
		return errors.New("recreate-missing-state-from enabled for a non-archive node")
	}
	if err := c.Init.Validate(); err != nil {
		return err
	}
	if err := c.ParentChain.Validate(); err != nil {
		return err
	}
	if err := c.Node.Validate(); err != nil {
		return err
	}
	if err := c.Execution.Validate(); err != nil {
		return err
	}
	if c.Node.ExecutionRPCClient.URL == "self" || c.Node.ExecutionRPCClient.URL == "self-auth" {
		if c.Node.Sequencer || c.Node.BatchPoster.Enable || c.Node.BlockValidator.Enable {
			return errors.New("sequencing, validation and batch-posting are currently not supported when connecting to an execution client over RPC")
		}
		if !c.Node.RPCServer.Enable {
			return errors.New("consensus and execution are configured to communicate over rpc but consensus node has not enabled rpc server")
		}
		if !c.Execution.RPCServer.Enable {
			return errors.New("consensus and execution are configured to communicate over rpc but execution node has not enabled rpc server")
		}
		if c.Execution.ConsensusRPCClient.URL != c.Node.ExecutionRPCClient.URL {
			return errors.New("consensus and execution are configured to communicate over rpc but execution node has consensusRPCClient url not equal to that of execution (self or self-auth)")
		}
		if c.WS.Addr == "" {
			return errors.New("consensus and execution are configured to communicate over rpc but websocket is not enabled")
		}
	} else if c.Node.ExecutionRPCClient.URL != "" {
		if c.Node.Sequencer || c.Node.BatchPoster.Enable || c.Node.BlockValidator.Enable {
			return errors.New("sequencing, validation and batch-posting are currently not supported when connecting to an execution client over RPC")
		}
	} else if c.Execution.ConsensusRPCClient.URL != "" {
		return errors.New("consensus is connecting directly to execution but execution is connecting to consensus over an rpc- invalid case")
	}
	if err := c.BlocksReExecutor.Validate(); err != nil {
		return err
	}
	if c.Node.ValidatorRequired() && (c.Execution.Caching.StateScheme == rawdb.PathScheme) {
		return errors.New("path cannot be used as execution.caching.state-scheme when validator is required")
	}
	return c.Persistent.Validate()
}

func (c *NodeConfig) GetReloadInterval() time.Duration {
	return c.Conf.ReloadInterval
}

func NodeConfigAddOptions(f *pflag.FlagSet) {
	genericconf.ConfConfigAddOptions("conf", f)
	arbnode.ConfigAddOptions("node", f, true, true)
	gethexec.ConfigAddOptions("execution", f)
	valnode.ValidationConfigAddOptions("validation", f)
	conf.L1ConfigAddOptions("parent-chain", f)
	conf.L2ConfigAddOptions("chain", f)
	f.String("log-level", NodeConfigDefault.LogLevel, "log level, valid values are CRIT, ERROR, WARN, INFO, DEBUG, TRACE")
	f.String("log-type", NodeConfigDefault.LogType, "log type (plaintext or json)")
	genericconf.FileLoggingConfigAddOptions("file-logging", f)
	conf.PersistentConfigAddOptions("persistent", f)
	genericconf.HTTPConfigAddOptions("http", f)
	genericconf.WSConfigAddOptions("ws", f)
	genericconf.IPCConfigAddOptions("ipc", f)
	genericconf.AuthRPCConfigAddOptions("auth", f)
	genericconf.GraphQLConfigAddOptions("graphql", f)
	f.Bool("metrics", NodeConfigDefault.Metrics, "enable metrics")
	genericconf.MetricsServerAddOptions("metrics-server", f)
	f.Bool("pprof", NodeConfigDefault.PProf, "enable pprof")
	genericconf.PProfAddOptions("pprof-cfg", f)

	conf.InitConfigAddOptions("init", f)
	genericconf.RpcConfigAddOptions("rpc", f)
	blocksreexecutor.ConfigAddOptions("blocks-reexecutor", f)
	f.Bool("ensure-rollup-deployment", NodeConfigDefault.EnsureRollupDeployment, "before starting the node, wait until the transaction that deployed rollup is finalized")
}

func ParseNode(ctx context.Context, args []string) (*NodeConfig, *genericconf.WalletConfig, error) {
	f := pflag.NewFlagSet("", pflag.ContinueOnError)

	NodeConfigAddOptions(f)

	k, err := confighelpers.BeginCommonParse(f, args)
	if err != nil {
		return nil, nil, err
	}

	l2ChainId := k.Int64("chain.id")
	l2ChainName := k.String("chain.name")
	l2ChainInfoFiles := k.Strings("chain.info-files")
	l2ChainInfoJson := k.String("chain.info-json")
	l2GenesisJsonFile := k.String("init.genesis-json-file")
	// #nosec G115
	err = applyChainParameters(k, uint64(l2ChainId), l2ChainName, l2ChainInfoFiles, l2ChainInfoJson, l2GenesisJsonFile)
	if err != nil {
		return nil, nil, err
	}

	err = confighelpers.ApplyOverrides(f, k)
	if err != nil {
		return nil, nil, err
	}

	if err = anytrust.FixKeysetCLIParsing("node.data-availability.rpc-aggregator.backends", k); err != nil {
		return nil, nil, err
	}
	if err = anytrust.FixKeysetCLIParsing("node.da.anytrust.rpc-aggregator.backends", k); err != nil {
		return nil, nil, err
	}

	var nodeConfig NodeConfig
	if err := confighelpers.EndCommonParse(k, &nodeConfig); err != nil {
		return nil, nil, err
	}

	// Migrate deprecated --node.data-availability.* to --node.da.anytrust.*
	nodeConfig.Node.MigrateDeprecatedConfig()

	// Don't print wallet passwords
	if nodeConfig.Conf.Dump {
		err = confighelpers.DumpConfig(k, map[string]interface{}{
			"node.batch-poster.parent-chain-wallet.password":    "",
			"node.batch-poster.parent-chain-wallet.private-key": "",
			"node.staker.parent-chain-wallet.password":          "",
			"node.staker.parent-chain-wallet.private-key":       "",
			"chain.dev-wallet.password":                         "",
			"chain.dev-wallet.private-key":                      "",
		})
		if err != nil {
			return nil, nil, err
		}
	}

	if nodeConfig.Persistent.Chain == "" {
		return nil, nil, errors.New("--persistent.chain not specified")
	}

	err = nodeConfig.ResolveDirectoryNames()
	if err != nil {
		return nil, nil, err
	}

	// Don't pass around wallet contents with normal configuration
	l2DevWallet := nodeConfig.Chain.DevWallet
	nodeConfig.Chain.DevWallet = genericconf.WalletConfigDefault

	if nodeConfig.Execution.Caching.Archive {
		nodeConfig.Node.MessagePruner.Enable = false
	}

	if nodeConfig.Execution.Caching.Archive && (!nodeConfig.Execution.TxIndexer.Enable || nodeConfig.Execution.TxIndexer.TxLookupLimit != 0) {
		log.Info("retaining ability to lookup full transaction history as archive mode is enabled")
		nodeConfig.Execution.TxIndexer.Enable = true
		nodeConfig.Execution.TxIndexer.TxLookupLimit = 0
	}

	err = nodeConfig.Validate()
	if err != nil {
		return nil, nil, err
	}
	return &nodeConfig, &l2DevWallet, nil
}

func applyChainParameters(k *koanf.Koanf, chainId uint64, chainName string, l2ChainInfoFiles []string, l2ChainInfoJson string, l2GenesisJsonFile string) error {
	chainInfo, err := chaininfo.ProcessChainInfo(chainId, chainName, l2ChainInfoFiles, l2ChainInfoJson)
	if err != nil {
		return err
	}
	var parentChainIsArbitrum bool
	if chainInfo.ParentChainIsArbitrum != nil {
		parentChainIsArbitrum = *chainInfo.ParentChainIsArbitrum
	} else {
		log.Warn("Chain info field parent-chain-is-arbitrum is missing, in the future this will be required", "chainId", chainInfo.ChainConfig.ChainID, "parentChainId", chainInfo.ParentChainId)
		_, err := chaininfo.ProcessChainInfo(chainInfo.ParentChainId, "", l2ChainInfoFiles, "")
		if err == nil {
			parentChainIsArbitrum = true
		}
	}
	chainDefaults := map[string]interface{}{
		"persistent.chain": chainInfo.ChainName,
		"chain.name":       chainInfo.ChainName,
		"chain.id":         chainInfo.ChainConfig.ChainID.Uint64(),
		"parent-chain.id":  chainInfo.ParentChainId,
	}
	// Only use chainInfo.SequencerUrl as default forwarding-target if sequencer is not enabled
	if !k.Bool("execution.sequencer.enable") && chainInfo.SequencerUrl != "" {
		chainDefaults["execution.forwarding-target"] = chainInfo.SequencerUrl
	}
	if chainInfo.SecondaryForwardingTarget != "" {
		chainDefaults["execution.secondary-forwarding-target"] = strings.Split(chainInfo.SecondaryForwardingTarget, ",")
	}
	if chainInfo.FeedUrl != "" {
		chainDefaults["node.feed.input.url"] = strings.Split(chainInfo.FeedUrl, ",")
	}
	if chainInfo.SecondaryFeedUrl != "" {
		chainDefaults["node.feed.input.secondary-url"] = strings.Split(chainInfo.SecondaryFeedUrl, ",")
	}
	if chainInfo.FeedSigned {
		chainDefaults["node.feed.input.verify.dangerous.accept-missing"] = false
	}
	if chainInfo.DasIndexUrl != "" {
		// Set defaults at the new AnyTrust config path only
		// Users of old --node.data-availability.* flags will be migrated
		chainDefaults["node.da.anytrust.enable"] = true
		chainDefaults["node.da.anytrust.rest-aggregator.enable"] = true
		chainDefaults["node.da.anytrust.rest-aggregator.online-url-list"] = chainInfo.DasIndexUrl
	} else if chainInfo.ChainConfig.ArbitrumChainParams.DataAvailabilityCommittee {
		chainDefaults["node.da.anytrust.enable"] = true
	}
	if !chainInfo.HasGenesisState && l2GenesisJsonFile == "" {
		chainDefaults["init.empty"] = true
	}
	if parentChainIsArbitrum {
		l2MaxTxSize := gethexec.DefaultSequencerConfig.MaxTxDataSize
		bufferSpace := 5000
		if l2MaxTxSize < bufferSpace*2 {
			return fmt.Errorf("not enough room in parent chain max tx size %v for bufferSpace %v * 2", l2MaxTxSize, bufferSpace)
		}
		safeBatchSize := l2MaxTxSize - bufferSpace
		chainDefaults["node.batch-poster.max-calldata-batch-size"] = safeBatchSize
		chainDefaults["execution.sequencer.max-tx-data-size"] = safeBatchSize - bufferSpace
		// Arbitrum chains produce blocks more quickly, so the inbox reader should read more blocks at once.
		// Even if this is too large, on error the inbox reader will reset its query size down to the default.
		chainDefaults["node.inbox-reader.max-blocks-to-read"] = 10_000
	}
	if chainInfo.DasIndexUrl != "" {
		chainDefaults["node.batch-poster.max-calldata-batch-size"] = 1_000_000
	}
	// 0 is default for any chain unless specified in the chain_defaults
	chainDefaults["node.transaction-streamer.track-block-metadata-from"] = chainInfo.TrackBlockMetadataFrom
	chainDefaults["node.block-metadata-fetcher.source.url"] = chainInfo.BlockMetadataUrl
	if chainInfo.TrackBlockMetadataFrom > 0 && chainInfo.BlockMetadataUrl != "" {
		chainDefaults["node.block-metadata-fetcher.enable"] = true
	}

	err = k.Load(confmap.Provider(chainDefaults, "."), nil)
	if err != nil {
		return err
	}
	return nil
}

type ConsensusNodeConfigFetcher struct {
	*genericconf.LiveConfig[*NodeConfig]
}

func (f *ConsensusNodeConfigFetcher) Get() *arbnode.Config {
	return &f.LiveConfig.Get().Node
}

type ExecutionNodeConfigFetcher struct {
	*genericconf.LiveConfig[*NodeConfig]
}

func (f *ExecutionNodeConfigFetcher) Get() *gethexec.Config {
	return &f.LiveConfig.Get().Execution
}
