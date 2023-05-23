package rpcclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync/atomic"
	"time"

	flag "github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/util/signature"
)

type ClientConfig struct {
	URL            string        `koanf:"url"`
	JWTSecret      string        `koanf:"jwtsecret"`
	Timeout        time.Duration `koanf:"timeout" reload:"hot"`
	Retries        uint          `koanf:"retries" reload:"hot"`
	ConnectionWait time.Duration `koanf:"connection-wait"`
	ArgLogLimit    uint          `koanf:"arg-log-limit" reload:"hot"`
	RetryErrors    string        `koanf:"retry-errors" reload:"hot"`
}

type ClientConfigFetcher func() *ClientConfig

var TestClientConfig = ClientConfig{
	URL:       "self",
	JWTSecret: "",
}

var DefaultClientConfig = ClientConfig{
	URL:         "self-auth",
	JWTSecret:   "",
	ArgLogLimit: 2048,
}

func RPCClientAddOptions(prefix string, f *flag.FlagSet, defaultConfig *ClientConfig) {
	f.String(prefix+".url", defaultConfig.URL, "url of server, use self for loopback websocket, self-auth for loopback with authentication")
	f.String(prefix+".jwtsecret", defaultConfig.JWTSecret, "path to file with jwtsecret for validation - ignored if url is self or self-auth")
	f.Duration(prefix+".connection-wait", defaultConfig.ConnectionWait, "how long to wait for initial connection")
	f.Duration(prefix+".timeout", defaultConfig.Timeout, "per-response timeout (0-disabled)")
	f.Uint(prefix+".arg-log-limit", defaultConfig.ArgLogLimit, "limit size of arguments in log entries")
	f.Uint(prefix+".retries", defaultConfig.Retries, "number of retries in case of failure(0 mean one attempt)")
	f.String(prefix+".retry-errors", defaultConfig.RetryErrors, "Errors matching this regular expression are automatically retried")
}

type RpcClient struct {
	config    ClientConfigFetcher
	client    *rpc.Client
	autoStack *node.Node
	logId     uint64
}

func NewRpcClient(config ClientConfigFetcher, stack *node.Node) *RpcClient {
	return &RpcClient{
		config:    config,
		autoStack: stack,
	}
}

func (c *RpcClient) Close() {
	c.client.Close()
}

func limitString(limit int, str string) string {
	if limit == 0 || len(str) <= limit {
		return str
	}
	prefix := str[:limit/2-1]
	postfix := str[len(str)-limit/2+1:]
	return fmt.Sprintf("%v..%v", prefix, postfix)
}

func logArgs(limit int, args ...interface{}) string {
	res := "["
	for i, arg := range args {
		marshalled, err := json.Marshal(arg)
		if err != nil {
			res += "\"CANNOT MARSHALL:" + limitString(limit, err.Error()) + "\""
		} else {
			res += limitString(limit, string(marshalled))
		}
		if i < len(args)-1 {
			res += ", "
		}
	}
	res += "]"
	return res
}

func (c *RpcClient) CallContext(ctx_in context.Context, result interface{}, method string, args ...interface{}) error {
	if c.client == nil {
		return errors.New("not connected")
	}
	logId := atomic.AddUint64(&c.logId, 1)
	log.Trace("sending RPC request", "method", method, "logId", logId, "args", logArgs(int(c.config().ArgLogLimit), args...))
	var err error
	for i := 0; i < int(c.config().Retries)+1; i++ {
		if ctx_in.Err() != nil {
			return ctx_in.Err()
		}
		var ctx context.Context
		var cancelCtx context.CancelFunc
		timeout := c.config().Timeout
		if timeout > 0 {
			ctx, cancelCtx = context.WithTimeout(ctx_in, timeout)
		} else {
			ctx, cancelCtx = context.WithCancel(ctx_in)
		}
		err = c.client.CallContext(ctx, result, method, args...)
		cancelCtx()
		logger := log.Trace
		limit := int(c.config().ArgLogLimit)
		if err != nil && err.Error() != "already known" {
			logger = log.Info
			limit = 0
		}
		logger("rpc response", "method", method, "logId", logId, "err", err, "result", limitString(limit, fmt.Sprintf("%+v", result)), "attempt", i, "args", logArgs(limit, args...))
		if err == nil {
			return nil
		}
		if errors.Is(err, context.DeadlineExceeded) {
			continue
		}
		retryErrors := c.config().RetryErrors
		if retryErrors != "" {
			match, regexErr := regexp.MatchString(retryErrors, err.Error())
			if regexErr != nil {
				log.Warn("rpcclient: bad value for retry-error. Not retrying.", "err", err, "value", retryErrors)
			}
			if match {
				continue
			}
		}
		return err
	}
	return err
}

func (c *RpcClient) BatchCallContext(ctx context.Context, b []rpc.BatchElem) error {
	return c.client.BatchCallContext(ctx, b)
}

func (c *RpcClient) EthSubscribe(ctx context.Context, channel interface{}, args ...interface{}) (*rpc.ClientSubscription, error) {
	return c.client.EthSubscribe(ctx, channel, args...)
}

func (c *RpcClient) Start(ctx_in context.Context) error {
	url := c.config().URL
	jwtPath := c.config().JWTSecret
	if url == "self" {
		if c.autoStack == nil {
			return errors.New("self not supported for this connection")
		}
		url = c.autoStack.WSEndpoint()
		jwtPath = ""
	} else if url == "self-auth" {
		if c.autoStack == nil {
			return errors.New("self-auth not supported for this connection")
		}
		url = c.autoStack.WSAuthEndpoint()
		jwtPath = c.autoStack.JWTPath()
	} else if url == "" {
		return errors.New("no url provided for this connection")
	}
	var jwt *common.Hash
	if jwtPath != "" {
		var err error
		jwt, err = signature.LoadSigningKey(jwtPath)
		if err != nil {
			return err
		}
	}
	connTimeout := time.After(c.config().ConnectionWait)
	for {
		var ctx context.Context
		var cancelCtx context.CancelFunc
		timeout := c.config().Timeout
		if timeout > 0 {
			ctx, cancelCtx = context.WithTimeout(ctx_in, timeout)
		} else {
			ctx, cancelCtx = context.WithCancel(ctx_in)
		}
		var err error
		var client *rpc.Client
		if jwt == nil {
			client, err = rpc.DialContext(ctx, url)
		} else {
			client, err = rpc.DialOptions(ctx, url, rpc.WithHTTPAuth(node.NewJWTAuth([32]byte(*jwt))))
		}
		cancelCtx()
		if err == nil {
			c.client = client
			return nil
		}
		if strings.Contains(err.Error(), "parse") ||
			strings.Contains(err.Error(), "malformed") {
			return fmt.Errorf("%w: url %s", err, url)
		}
		select {
		case <-connTimeout:
			return fmt.Errorf("timeout trying to connect lastError: %w", err)
		case <-time.After(time.Second):
		}
	}
}
