package rpcclient

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	flag "github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/util/signature"
)

type ClientConfig struct {
	URL            string        `koanf:"url"`
	JWTSecret      string        `koanf:"jwtsecret"`
	Timeout        time.Duration `koanf:"timeout"`
	Retries        uint          `koanf:"retries"`
	ConnectionWait time.Duration `koanf:"connection-wait"`
	TraceLogLimit  uint          `koanf:"log-limit"`
}

var TestClientConfig = ClientConfig{
	URL:           "auto",
	JWTSecret:     "",
	TraceLogLimit: 2048,
}

var DefaultClientConfig = ClientConfig{
	URL:           "auto-auth",
	JWTSecret:     "",
	TraceLogLimit: 2048,
}

func RPCClientAddOptions(prefix string, f *flag.FlagSet, defaultConfig *ClientConfig) {
	f.String(prefix+".url", defaultConfig.URL, "url of server, use auto for loopback websocket, auto-auth for loopback with authentication")
	f.String(prefix+".jwtsecret", defaultConfig.JWTSecret, "path to file with jwtsecret for validation - ignored if url is auto or auto-auth")
	f.Duration(prefix+"connection-wait", defaultConfig.ConnectionWait, "how long to wait for initial connection")
	f.Duration(prefix+"timeout", defaultConfig.Timeout, "per-response timeout (0-disabled)")
	f.Uint(prefix+".log-limit", defaultConfig.TraceLogLimit, "limit size of log entries")
	f.Uint(prefix+".retries", defaultConfig.Retries, "number of retries in case of failure(0 mean one attempt)")
}

type RpcClient struct {
	config    *ClientConfig
	client    *rpc.Client
	autoStack *node.Node
	logId     uint64
}

func NewRpcClient(config *ClientConfig, stack *node.Node) *RpcClient {
	return &RpcClient{
		config:    config,
		autoStack: stack,
	}
}

func (c *RpcClient) Close() {
	c.client.Close()
}

func (c *RpcClient) CallContext(ctx_in context.Context, result interface{}, method string, args ...interface{}) error {
	if c.client == nil {
		return errors.New("not connected")
	}
	logId := atomic.AddUint64(&c.logId, 1)
	log.Trace("sending RPC request", "method", method, "logId", logId)
	var err error
	for i := 0; i < int(c.config.Retries)+1; i++ {
		var ctx context.Context
		var cancelCtx context.CancelFunc
		if c.config.Timeout > 0 {
			ctx, cancelCtx = context.WithTimeout(ctx_in, c.config.Timeout)
		} else {
			ctx, cancelCtx = context.WithCancel(ctx_in)
		}
		err = c.client.CallContext(ctx, result, method, args...)
		cancelCtx()
		logger := log.Trace
		if err != nil && err.Error() != "already known" {
			logger = log.Info
		}
		logger("rpc response", "method", method, "logId", logId, "result", result, "attempt", i)
		if !errors.Is(err, context.DeadlineExceeded) {
			return err
		}
	}
	return err
}

func (c *RpcClient) Start(ctx_in context.Context) error {
	url := c.config.URL
	jwtPath := c.config.JWTSecret
	if url == "auto" {
		url = c.autoStack.WSEndpoint()
		jwtPath = ""
	} else if url == "auto-auth" {
		url, jwtPath = c.autoStack.AuthEndpoint(true)
	}
	var jwtBytes []byte
	if jwtPath != "" {
		jwtHash, err := signature.LoadSigningKey(jwtPath)
		if err != nil {
			return err
		}
		jwtBytes = jwtHash.Bytes()
	}
	timeout := time.After(c.config.ConnectionAttempts)
	for {
		var ctx context.Context
		var cancelCtx context.CancelFunc
		if c.config.Timeout > 0 {
			ctx, cancelCtx = context.WithTimeout(ctx_in, c.config.Timeout)
		} else {
			ctx, cancelCtx = context.WithCancel(ctx_in)
		}
		var err error
		var client *rpc.Client
		if len(jwtBytes) == 0 {
			client, err = rpc.DialWebsocket(ctx, url, "")
		} else {
			client, err = rpc.DialWebsocketJWT(ctx, url, "", jwtBytes)
		}
		cancelCtx()
		if err == nil {
			c.client = client
			return nil
		}
		if strings.Contains(err.Error(), "parse") {
			return fmt.Errorf("%w: url %s", err, url)
		}
		select {
		case <-timeout:
			return fmt.Errorf("timeout trying to connect lastError: %w", err)
		case <-time.After(time.Second):
		}
	}
}
