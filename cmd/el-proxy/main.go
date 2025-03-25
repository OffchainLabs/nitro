package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	flag "github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/metrics/exp"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/cmd/util/confighelpers"
	"github.com/offchainlabs/nitro/timeboost"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

func printSampleUsage(name string) {
	fmt.Printf("Sample usage: %s --help \n", name)
}

func main() {
	os.Exit(mainImpl())
}

// Checks metrics and PProf flag, runs them if enabled.
// Note: they are separate so one can enable/disable them as they wish, the only
// requirement is that they can't run on the same address and port.
func startMetrics(cfg *ExpressLaneProxyConfig) error {
	mAddr := fmt.Sprintf("%v:%v", cfg.MetricsServer.Addr, cfg.MetricsServer.Port)
	pAddr := fmt.Sprintf("%v:%v", cfg.PprofCfg.Addr, cfg.PprofCfg.Port)
	if cfg.Metrics && !metrics.Enabled() {
		return fmt.Errorf("metrics must be enabled via command line by adding --metrics, json config has no effect")
	}
	if cfg.Metrics && cfg.PProf && mAddr == pAddr {
		return fmt.Errorf("metrics and pprof cannot be enabled on the same address:port: %s", mAddr)
	}
	if cfg.Metrics {
		go metrics.CollectProcessMetrics(time.Second)
		exp.Setup(fmt.Sprintf("%v:%v", cfg.MetricsServer.Addr, cfg.MetricsServer.Port))
	}
	if cfg.PProf {
		genericconf.StartPprof(pAddr)
	}
	return nil
}

type ExpressLaneProxy struct {
	stopwaiter.StopWaiter
	config *ExpressLaneProxyConfig
}

func NewExpressLaneProxy(
	ctx context.Context,
	config *ExpressLaneProxyConfig,
	stack *node.Node,
) *ExpressLaneProxy {
	elProxy := &ExpressLaneProxy{
		config: config,
	}

	elAPIs := []rpc.API{{
		Namespace: "eth",
		Version:   "1.0",
		Service:   elProxy,
		Public:    true,
	}}

	stack.RegisterAPIs(elAPIs)
	return elProxy
}

var ErrorInternalConnectionError = errors.New("internal connection error")

func GetClientFromURL(ctx context.Context, rawUrl string, transport *http.Transport) (*rpc.Client, error) {
	u, err := url.Parse(rawUrl)
	if err != nil {
		log.Error("error getting client from url", "error", err)
		return nil, ErrorInternalConnectionError
	}

	var rpcClient *rpc.Client
	switch u.Scheme {
	case "http", "https":
		if transport != nil {
			client := &http.Client{
				Transport: transport,
			}
			rpcClient, err = rpc.DialOptions(ctx, rawUrl, rpc.WithHTTPClient(client))
		} else {
			rpcClient, err = rpc.DialHTTP(rawUrl)
		}
	case "ws", "wss":
		rpcClient, err = rpc.DialWebsocket(ctx, rawUrl, "")
	default:
		log.Error("no known transport", "scheme", u.Scheme, "url", rawUrl)
		return nil, ErrorInternalConnectionError
	}
	if err != nil {
		log.Error("error connecting to client", "error", err, "url", rawUrl)
		return nil, ErrorInternalConnectionError
	}
	return rpcClient, nil
}

func (p *ExpressLaneProxy) SendRawTransaction(ctx context.Context, input hexutil.Bytes) (common.Hash, error) {

	wrapper := timeboost.JsonExpressLaneSubmission{}

	client, err := GetClientFromURL(ctx, p.config.ExpressLaneURL, nil)
	if err != nil {
		return common.Hash{}, err
	}

	err = client.CallContext(ctx, nil, "timeboost_sendExpressLaneTransaction", &wrapper)
	return common.Hash{}, err
}

func mainImpl() int {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	args := os.Args[1:]
	expressLaneProxyConfig, err := parseExpressLaneProxyArgs(ctx, args)
	if err != nil {
		confighelpers.PrintErrorAndExit(err, printSampleUsage)
		panic(err)
	}
	stackConf := DefaultExpressLaneProxyStackConfig
	stackConf.DataDir = "" // ephemeral
	expressLaneProxyConfig.HTTP.Apply(&stackConf)
	expressLaneProxyConfig.WS.Apply(&stackConf)
	expressLaneProxyConfig.IPC.Apply(&stackConf)
	stackConf.P2P.ListenAddr = ""
	stackConf.P2P.NoDial = true
	stackConf.P2P.NoDiscovery = true
	vcsRevision, strippedRevision, vcsTime := confighelpers.GetVersion()
	stackConf.Version = strippedRevision

	pathResolver := func(workdir string) func(string) string {
		if workdir == "" {
			workdir, err = os.Getwd()
			if err != nil {
				log.Warn("Failed to get workdir", "err", err)
			}
		}
		return func(path string) string {
			if filepath.IsAbs(path) {
				return path
			}
			return filepath.Join(workdir, path)
		}
	}

	err = genericconf.InitLog(expressLaneProxyConfig.LogType, expressLaneProxyConfig.LogLevel, &expressLaneProxyConfig.FileLogging, pathResolver(expressLaneProxyConfig.Persistent.LogDir))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing logging: %v\n", err)
		return 1
	}
	if stackConf.JWTSecret == "" && stackConf.AuthAddr != "" {
		filename := pathResolver(expressLaneProxyConfig.Persistent.GlobalConfig)("jwtsecret")
		if err := genericconf.TryCreatingJWTSecret(filename); err != nil {
			log.Error("Failed to prepare jwt secret file", "err", err)
			return 1
		}
		stackConf.JWTSecret = filename
	}

	liveNodeConfig := genericconf.NewLiveConfig[*ExpressLaneProxyConfig](args, expressLaneProxyConfig, parseExpressLaneProxyArgs)
	liveNodeConfig.SetOnReloadHook(func(oldCfg *ExpressLaneProxyConfig, newCfg *ExpressLaneProxyConfig) error {

		return genericconf.InitLog(newCfg.LogType, newCfg.LogLevel, &newCfg.FileLogging, pathResolver(expressLaneProxyConfig.Persistent.LogDir))
	})

	if err := startMetrics(expressLaneProxyConfig); err != nil {
		log.Error("Error starting metrics", "error", err)
		return 1
	}

	fatalErrChan := make(chan error, 10)

	// TODO start any proxy stuff here
	log.Info("Running Arbitrum Express Lane Proxy", "revision", vcsRevision, "vcs.time", vcsTime)
	stack, err := node.New(&stackConf)
	if err != nil {
		flag.Usage()
		log.Crit("failed to initialize geth stack", "err", err)
	}
	proxy := NewExpressLaneProxy(ctx, expressLaneProxyConfig, stack)
	_ = proxy
	err = stack.Start()
	if err != nil {
		log.Error("error", "err", err)
		return 1
	}

	defer stack.Close()

	liveNodeConfig.Start(ctx)
	defer liveNodeConfig.StopAndWait()

	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)

	exitCode := 0
	select {
	case err := <-fatalErrChan:
		log.Error("shutting down due to fatal error", "err", err)
		defer log.Error("shut down due to fatal error", "err", err)
		exitCode = 1
	case <-sigint:
		log.Info("shutting down because of sigint")
	}
	// cause future ctrl+c's to panic
	close(sigint)
	return exitCode
}

func parseExpressLaneProxyArgs(ctx context.Context, args []string) (*ExpressLaneProxyConfig, error) {
	f := flag.NewFlagSet("", flag.ContinueOnError)

	ExpressLaneProxyConfigAddOptions(f)

	k, err := confighelpers.BeginCommonParse(f, args)
	if err != nil {
		return nil, err
	}

	err = confighelpers.ApplyOverrides(f, k)
	if err != nil {
		return nil, err
	}

	var cfg ExpressLaneProxyConfig
	if err := confighelpers.EndCommonParse(k, &cfg); err != nil {
		return nil, err
	}

	// Don't print wallet passwords
	if cfg.Conf.Dump {
		err = confighelpers.DumpConfig(k, map[string]interface{}{
			"l1.wallet.password":        "",
			"l1.wallet.private-key":     "",
			"l2.dev-wallet.password":    "",
			"l2.dev-wallet.private-key": "",
		})
		if err != nil {
			return nil, err
		}
	}

	// Don't pass around wallet contents with normal configuration
	err = cfg.Validate()
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}
