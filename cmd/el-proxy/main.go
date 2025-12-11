// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

// el-proxy is an example implementation of a Timeboost Express Lane proxy
// and should only be used for testing purposes. It listens for
// eth_sendRawTransaction messages, wraps them, and forwards them to
// an endpoint implementing timeboost_sendExpressLaneTransaction.
// It also forwards other methods needed by tools like cast to build
// and check on the transaction to an RPC url which may be different to
// the express lane url.

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/metrics/exp"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/cmd/util"
	"github.com/offchainlabs/nitro/cmd/util/confighelpers"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/solgen/go/express_lane_auctiongen"
	"github.com/offchainlabs/nitro/timeboost"
	"github.com/offchainlabs/nitro/util/signature"
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
	config              *ExpressLaneProxyConfig
	roundTimingInfo     timeboost.RoundTimingInfo
	auctionContractAddr common.Address
	expressLaneTracker  *gethexec.ExpressLaneTracker
	dataSignerFunc      signature.DataSignerFunc
}

type HeaderProviderAdapter struct {
	*ethclient.Client
}

func (a *HeaderProviderAdapter) HeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Header, error) {
	return a.Client.HeaderByNumber(ctx, big.NewInt((int64)(number)))
}

func NewExpressLaneProxy(
	ctx context.Context,
	config *ExpressLaneProxyConfig,
	stack *node.Node,
) (*ExpressLaneProxy, error) {
	client, err := GetClientFromURL(ctx, config.RPCURL, nil)
	if err != nil {
		return nil, err
	}
	arbClient := ethclient.NewClient(client)

	if !common.IsHexAddress(config.AuctionContractAddress) {
		return nil, fmt.Errorf("invalid auction-contract-address \"%v\"", config.AuctionContractAddress)
	}
	auctionContractAddr := common.HexToAddress(config.AuctionContractAddress)
	auctionContract, err := express_lane_auctiongen.NewExpressLaneAuction(auctionContractAddr, arbClient)
	if err != nil {
		return nil, err
	}

	roundTimingInfo, err := gethexec.GetRoundTimingInfo(auctionContract)
	if err != nil {
		return nil, err
	}

	expressLaneTracker, err := gethexec.NewExpressLaneTracker(
		*roundTimingInfo,
		time.Millisecond*250,
		&HeaderProviderAdapter{arbClient},
		auctionContract,
		auctionContractAddr,
		&params.ChainConfig{ChainID: big.NewInt(config.ChainId)},
		config.MaxTxDataSize,
		0,
	)
	if err != nil {
		return nil, fmt.Errorf("error creating express lane tracker: %w", err)
	}

	_, dataSignerFunc, err := util.OpenWallet("el-proxy", &config.Wallet, big.NewInt(config.ChainId))
	if err != nil {
		return nil, fmt.Errorf("error opening wallet: %w", err)
	}

	elProxy := &ExpressLaneProxy{
		config:              config,
		roundTimingInfo:     *roundTimingInfo,
		auctionContractAddr: auctionContractAddr,
		expressLaneTracker:  expressLaneTracker,
		dataSignerFunc:      dataSignerFunc,
	}

	elAPIs := []rpc.API{{
		Namespace: "eth",
		Version:   "1.0",
		Service:   elProxy,
		Public:    true,
	}}

	stack.RegisterAPIs(elAPIs)
	return elProxy, nil
}

func (p *ExpressLaneProxy) Start(ctx context.Context) {
	p.StopWaiter.Start(ctx, p)
	p.expressLaneTracker.Start(ctx)
}

func (p *ExpressLaneProxy) StopAndWait() {
	p.StopWaiter.StopAndWait()
	p.expressLaneTracker.StopAndWait()
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

func (p *ExpressLaneProxy) buildSignature(data []byte) ([]byte, error) {
	prefixedData := crypto.Keccak256(append([]byte(fmt.Sprintf("\x19Ethereum Signed Message:\n%d", len(data))), data...))
	signature, err := p.dataSignerFunc(prefixedData)
	if err != nil {
		return nil, err
	}
	return signature, nil
}

func (p *ExpressLaneProxy) SendRawTransaction(ctx context.Context, input hexutil.Bytes) (common.Hash, error) {
	roundNumber := p.roundTimingInfo.RoundNumber()

	wrapper := timeboost.JsonExpressLaneSubmission{
		ChainId:                (*hexutil.Big)(big.NewInt(p.config.ChainId)),
		Round:                  (hexutil.Uint64)(roundNumber),
		AuctionContractAddress: p.auctionContractAddr,
		Transaction:            input,
		SequenceNumber:         (hexutil.Uint64)(gethexec.DontCareSequence),
		Signature:              []byte{}, // It is set below
	}
	goWrapper, err := timeboost.JsonSubmissionToGo(&wrapper)
	if err != nil {
		return common.Hash{}, fmt.Errorf("Error converting submission: %w", err)
	}
	signableMsg, err := goWrapper.ToMessageBytes()
	if err != nil {
		return common.Hash{}, fmt.Errorf("Error serializing signable msg: %w", err)
	}
	sig, err := p.buildSignature(signableMsg)
	if err != nil {
		return common.Hash{}, fmt.Errorf("Error signing msg: %w", err)
	}
	wrapper.Signature = sig

	client, err := GetClientFromURL(ctx, p.config.ExpressLaneURL, nil)
	if err != nil {
		return common.Hash{}, fmt.Errorf("Error getting client: %w", err)
	}

	log.Info("Sending timeboost_sendExpressLaneTransaction", "round", roundNumber, "txHash", goWrapper.Transaction.Hash().Hex())
	err = client.CallContext(ctx, nil, "timeboost_sendExpressLaneTransaction", &wrapper)
	if err != nil {
		return common.Hash{}, fmt.Errorf("Error forwarding msg: %w", err)
	}

	return goWrapper.Transaction.Hash(), nil
}

// We need to proxy some other methods for tools like cast to use when building txs.

func (p *ExpressLaneProxy) ChainId(_ context.Context) hexutil.Uint64 {
	chainId := p.config.ChainId
	// #nosec G115
	return (hexutil.Uint64)(chainId)
}

func (p *ExpressLaneProxy) GetTransactionCount(ctx context.Context, address common.Address, blockNumOrHash rpc.BlockNumberOrHash) (hexutil.Uint64, error) {
	client, err := GetClientFromURL(ctx, p.config.RPCURL, nil)
	if err != nil {
		return 0, err
	}

	var result hexutil.Uint64
	err = client.CallContext(ctx, &result, "eth_getTransactionCount", address, blockNumOrHash)
	return result, err
}

func (p *ExpressLaneProxy) FeeHistory(ctx context.Context, blockCount hexutil.Uint64, lastBlock rpc.BlockNumber, rewardPercentiles []float64) (json.RawMessage, error) {
	var result json.RawMessage

	client, err := GetClientFromURL(ctx, p.config.RPCURL, nil)
	if err != nil {
		return result, err
	}

	err = client.CallContext(ctx, &result, "eth_feeHistory", blockCount, lastBlock, rewardPercentiles)
	return result, err
}

func (p *ExpressLaneProxy) BlockNumber(ctx context.Context) (uint64, error) {
	client, err := GetClientFromURL(ctx, p.config.RPCURL, nil)
	if err != nil {
		return 0, err
	}

	return ethclient.NewClient(client).BlockNumber(ctx)
}

func (p *ExpressLaneProxy) GetBlockByNumber(ctx context.Context, blockNum *rpc.BlockNumber, includeTxData bool) (json.RawMessage, error) {
	var result json.RawMessage

	client, err := GetClientFromURL(ctx, p.config.RPCURL, nil)
	if err != nil {
		return result, err
	}

	err = client.CallContext(ctx, &result, "eth_getBlockByNumber", blockNum, includeTxData)
	return result, err
}

func (p *ExpressLaneProxy) GetTransactionReceipt(ctx context.Context, txHash hexutil.Bytes, opts *json.RawMessage) (json.RawMessage, error) {
	log.Debug("Received eth_getTransactionReceipt", "txHash", txHash)

	var result json.RawMessage
	client, err := GetClientFromURL(ctx, p.config.RPCURL, nil)
	if err != nil {
		return result, err
	}

	err = client.CallContext(ctx, &result, "eth_getTransactionReceipt", txHash, opts)
	return result, err

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

	log.Info("Running Arbitrum Express Lane Proxy", "revision", vcsRevision, "vcs.time", vcsTime)
	stack, err := node.New(&stackConf)
	if err != nil {
		pflag.Usage()
		log.Crit("failed to initialize geth stack", "err", err)
	}
	proxy, err := NewExpressLaneProxy(ctx, expressLaneProxyConfig, stack)
	if err != nil {
		log.Error("error", "err", err)
		return 1
	}

	err = stack.Start()
	if err != nil {
		log.Error("error", "err", err)
		return 1
	}
	defer stack.Close()

	proxy.Start(ctx)
	defer proxy.StopAndWait()

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
	f := pflag.NewFlagSet("", pflag.ContinueOnError)

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
			"wallet.password":    "",
			"wallet.private-key": "",
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
