package valnode

import (
	"context"

	"github.com/offchainlabs/nitro/validator"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/offchainlabs/nitro/validator/server_api"
	"github.com/offchainlabs/nitro/validator/server_arb"
	"github.com/offchainlabs/nitro/validator/server_common"
	"github.com/offchainlabs/nitro/validator/server_jit"
	"github.com/offchainlabs/nitro/validator/valnode/redis"
	"github.com/spf13/pflag"

	arbredis "github.com/offchainlabs/nitro/validator/server_arb/redis"
)

type WasmConfig struct {
	RootPath               string   `koanf:"root-path"`
	EnableWasmrootsCheck   bool     `koanf:"enable-wasmroots-check"`
	AllowedWasmModuleRoots []string `koanf:"allowed-wasm-module-roots"`
}

func WasmConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.String(prefix+".root-path", DefaultWasmConfig.RootPath, "path to machine folders, each containing wasm files (machine.wavm.br, replay.wasm)")
	f.Bool(prefix+".enable-wasmroots-check", DefaultWasmConfig.EnableWasmrootsCheck, "enable check for compatibility of on-chain WASM module root with node")
	f.StringSlice(prefix+".allowed-wasm-module-roots", DefaultWasmConfig.AllowedWasmModuleRoots, "list of WASM module roots or mahcine base paths to match against on-chain WasmModuleRoot")
}

var DefaultWasmConfig = WasmConfig{
	RootPath:               "",
	EnableWasmrootsCheck:   true,
	AllowedWasmModuleRoots: []string{},
}

type Config struct {
	UseJit          bool                               `koanf:"use-jit"`
	ApiAuth         bool                               `koanf:"api-auth"`
	ApiPublic       bool                               `koanf:"api-public"`
	Arbitrator      server_arb.ArbitratorSpawnerConfig `koanf:"arbitrator" reload:"hot"`
	RedisExecRunner arbredis.ExecutionSpawnerConfig    `koanf:"redis-exec-runnner"`
	Jit             server_jit.JitSpawnerConfig        `koanf:"jit" reload:"hot"`
	Wasm            WasmConfig                         `koanf:"wasm"`
}

type ValidationConfigFetcher func() *Config

var DefaultValidationConfig = Config{
	UseJit:     true,
	Jit:        server_jit.DefaultJitSpawnerConfig,
	ApiAuth:    true,
	ApiPublic:  false,
	Arbitrator: server_arb.DefaultArbitratorSpawnerConfig,
	Wasm:       DefaultWasmConfig,
}

var TestValidationConfig = Config{
	UseJit:     true,
	Jit:        server_jit.DefaultJitSpawnerConfig,
	ApiAuth:    false,
	ApiPublic:  true,
	Arbitrator: server_arb.DefaultArbitratorSpawnerConfig,
	Wasm:       DefaultWasmConfig,
}

func ValidationConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".use-jit", DefaultValidationConfig.UseJit, "use jit for validation")
	f.Bool(prefix+".api-auth", DefaultValidationConfig.ApiAuth, "validate is an authenticated API")
	f.Bool(prefix+".api-public", DefaultValidationConfig.ApiPublic, "validate is a public API")
	server_arb.ArbitratorSpawnerConfigAddOptions(prefix+".arbitrator", f)
	server_jit.JitSpawnerConfigAddOptions(prefix+".jit", f)
	WasmConfigAddOptions(prefix+".wasm", f)
}

type ValidationNode struct {
	config     ValidationConfigFetcher
	arbSpawner *server_arb.ArbitratorSpawner
	jitSpawner *server_jit.JitSpawner

	redisConsumer    *redis.ValidationServer
	redisExecSpawner *arbredis.ExecutionSpawner
}

func EnsureValidationExposedViaAuthRPC(stackConf *node.Config) {
	found := false
	for _, module := range stackConf.AuthModules {
		if module == server_api.Namespace {
			found = true
			break
		}
	}
	if !found {
		stackConf.AuthModules = append(stackConf.AuthModules, server_api.Namespace)
	}
}

func CreateValidationNode(configFetcher ValidationConfigFetcher, stack *node.Node, fatalErrChan chan error) (*ValidationNode, error) {
	config := configFetcher()
	locator, err := server_common.NewMachineLocator(config.Wasm.RootPath)
	if err != nil {
		return nil, err
	}
	arbConfigFetcher := func() *server_arb.ArbitratorSpawnerConfig {
		return &configFetcher().Arbitrator
	}
	arbSpawner, err := server_arb.NewArbitratorSpawner(locator, arbConfigFetcher)
	if err != nil {
		return nil, err
	}
	var (
		serverAPI    *ExecServerAPI
		jitSpawner   *server_jit.JitSpawner
		redisSpawner *arbredis.ExecutionSpawner
	)
	if config.RedisExecRunner.Enabled() {
		es, err := arbredis.NewExecutionSpawner(&config.RedisExecRunner, arbSpawner)
		if err != nil {
			log.Error("creating redis execution spawner", "error", err)
		}
		redisSpawner = es
	}
	if config.UseJit {
		jitConfigFetcher := func() *server_jit.JitSpawnerConfig { return &configFetcher().Jit }
		var err error
		jitSpawner, err = server_jit.NewJitSpawner(locator, jitConfigFetcher, fatalErrChan)
		if err != nil {
			return nil, err
		}
		serverAPI = NewExecutionServerAPI(jitSpawner, arbSpawner, redisSpawner, arbConfigFetcher)
	} else {
		serverAPI = NewExecutionServerAPI(arbSpawner, arbSpawner, redisSpawner, arbConfigFetcher)
	}
	var redisConsumer *redis.ValidationServer
	redisValidationConfig := arbConfigFetcher().RedisValidationServerConfig
	if redisValidationConfig.Enabled() {
		redisConsumer, err = redis.NewValidationServer(&redisValidationConfig, arbSpawner)
		if err != nil {
			log.Error("Creating new redis validation server", "error", err)
		}
	}
	valAPIs := []rpc.API{{
		Namespace:     server_api.Namespace,
		Version:       "1.0",
		Service:       serverAPI,
		Public:        config.ApiPublic,
		Authenticated: config.ApiAuth,
	}}
	stack.RegisterAPIs(valAPIs)

	return &ValidationNode{
		config:           configFetcher,
		arbSpawner:       arbSpawner,
		jitSpawner:       jitSpawner,
		redisConsumer:    redisConsumer,
		redisExecSpawner: redisSpawner,
	}, nil
}

func (v *ValidationNode) Start(ctx context.Context) error {
	if err := v.arbSpawner.Start(ctx); err != nil {
		return err
	}
	if v.jitSpawner != nil {
		if err := v.jitSpawner.Start(ctx); err != nil {
			return err
		}
	}
	if v.redisConsumer != nil {
		v.redisConsumer.Start(ctx)
	}
	return nil
}

func (v *ValidationNode) GetExec() validator.ExecutionSpawner {
	return v.arbSpawner
}
