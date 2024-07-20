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
	URL                       string        `json:"url,omitempty" koanf:"url"`
	JWTSecret                 string        `json:"jwtsecret,omitempty" koanf:"jwtsecret"`
	Timeout                   time.Duration `json:"timeout,omitempty" koanf:"timeout" reload:"hot"`
	Retries                   uint          `json:"retries,omitempty" koanf:"retries" reload:"hot"`
	ConnectionWait            time.Duration `json:"connection-wait,omitempty" koanf:"connection-wait"`
	ArgLogLimit               uint          `json:"arg-log-limit,omitempty" koanf:"arg-log-limit" reload:"hot"`
	RetryErrors               string        `json:"retry-errors,omitempty" koanf:"retry-errors" reload:"hot"`
	RetryDelay                time.Duration `json:"retry-delay,omitempty" koanf:"retry-delay"`
	WebsocketMessageSizeLimit int64         `json:"websocket-message-size-limit,omitempty" koanf:"websocket-message-size-limit"`

	retryErrors *regexp.Regexp
}

func (c *ClientConfig) Validate() error {
	if c.RetryErrors == "" {
		c.retryErrors = nil
		return nil
	}
	var err error
	c.retryErrors, err = regexp.Compile(c.RetryErrors)
	return err
}

func (c *ClientConfig) UnmarshalJSON(data []byte) error {
	// Use DefaultClientConfig for default values when unmarshalling JSON
	*c = DefaultClientConfig
	type clientConfigWithoutCustomUnmarshal ClientConfig
	return json.Unmarshal(data, (*clientConfigWithoutCustomUnmarshal)(c))
}

type ClientConfigFetcher func() *ClientConfig

var TestClientConfig = ClientConfig{
	URL:                       "self",
	JWTSecret:                 "",
	WebsocketMessageSizeLimit: 256 * 1024 * 1024,
}

var DefaultClientConfig = ClientConfig{
	URL:                       "self-auth",
	JWTSecret:                 "",
	Retries:                   3,
	RetryErrors:               "websocket: close.*|dial tcp .*|.*i/o timeout|.*connection reset by peer|.*connection refused",
	ArgLogLimit:               2048,
	WebsocketMessageSizeLimit: 256 * 1024 * 1024,
}

func RPCClientAddOptions(prefix string, f *flag.FlagSet, defaultConfig *ClientConfig) {
	f.String(prefix+".url", defaultConfig.URL, "url of server, use self for loopback websocket, self-auth for loopback with authentication")
	f.String(prefix+".jwtsecret", defaultConfig.JWTSecret, "path to file with jwtsecret for validation - ignored if url is self or self-auth")
	f.Duration(prefix+".connection-wait", defaultConfig.ConnectionWait, "how long to wait for initial connection")
	f.Duration(prefix+".timeout", defaultConfig.Timeout, "per-response timeout (0-disabled)")
	f.Uint(prefix+".arg-log-limit", defaultConfig.ArgLogLimit, "limit size of arguments in log entries")
	f.Uint(prefix+".retries", defaultConfig.Retries, "number of retries in case of failure(0 mean one attempt)")
	f.String(prefix+".retry-errors", defaultConfig.RetryErrors, "Errors matching this regular expression are automatically retried")
	f.Duration(prefix+".retry-delay", defaultConfig.RetryDelay, "delay between retries")
	f.Int64(prefix+".websocket-message-size-limit", defaultConfig.WebsocketMessageSizeLimit, "websocket message size limit used by the RPC client. 0 means no limit")
}

type RpcClient struct {
	config    ClientConfigFetcher
	client    *rpc.Client
	autoStack *node.Node
	logId     atomic.Uint64
}

func NewRpcClient(config ClientConfigFetcher, stack *node.Node) *RpcClient {
	return &RpcClient{
		config:    config,
		autoStack: stack,
	}
}

func (c *RpcClient) Close() {
	if c.client != nil {
		c.client.Close()
	}
}

type limitedMarshal struct {
	limit int
	value any
}

func (m limitedMarshal) String() string {
	marshalled, err := json.Marshal(m.value)
	var str string
	if err != nil {
		str = "\"CANNOT MARSHALL: " + err.Error() + "\""
	} else {
		str = string(marshalled)
	}
	if m.limit == 0 || len(str) <= m.limit {
		return str
	}
	prefix := str[:m.limit/2-1]
	postfix := str[len(str)-m.limit/2+1:]
	return fmt.Sprintf("%v..%v", prefix, postfix)
}

type limitedArgumentsMarshal struct {
	limit int
	args  []any
}

func (m limitedArgumentsMarshal) String() string {
	res := "["
	for i, arg := range m.args {
		res += limitedMarshal{m.limit, arg}.String()
		if i < len(m.args)-1 {
			res += ", "
		}
	}
	res += "]"
	return res
}

var blobTxUnderpricedRegexp = regexp.MustCompile(`replacement transaction underpriced: new tx gas fee cap (\d*) <= (\d*) queued`)

// IsAlreadyKnownError returns true if the error appears to be an "already known" error.
// This check is based on the error's string form and is not precise.
func IsAlreadyKnownError(err error) bool {
	s := err.Error()
	if strings.Contains(s, "already known") {
		return true
	}
	// go-ethereum returns "replacement transaction underpriced" instead of "already known" for blob txs.
	// This is fixed in https://github.com/ethereum/go-ethereum/pull/29210
	// TODO: Once a new geth release is out with this fix, we can remove this check.
	matches := blobTxUnderpricedRegexp.FindSubmatch([]byte(s))
	if len(matches) == 3 {
		return string(matches[1]) == string(matches[2])
	}
	return false
}

func (c *RpcClient) CallContext(ctx_in context.Context, result interface{}, method string, args ...interface{}) error {
	if c.client == nil {
		return errors.New("not connected")
	}
	logId := c.logId.Add(1)
	log.Trace("sending RPC request", "method", method, "logId", logId, "args", limitedArgumentsMarshal{int(c.config().ArgLogLimit), args})
	var err error
	for i := 0; i < int(c.config().Retries)+1; i++ {
		retryDelay := c.config().RetryDelay
		if i > 0 && retryDelay > 0 {
			select {
			case <-ctx_in.Done():
				return ctx_in.Err()
			case <-time.After(retryDelay):
			}
		}
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
		if err != nil && !IsAlreadyKnownError(err) {
			logger = log.Info
		}
		logEntry := []interface{}{
			"method", method,
			"logId", logId,
			"err", err,
			"result", limitedMarshal{limit, result},
			"attempt", i,
			"args", limitedArgumentsMarshal{limit, args},
		}
		var dataErr rpc.DataError
		if errors.As(err, &dataErr) {
			logEntry = append(logEntry, "errorData", limitedMarshal{limit, dataErr.ErrorData()})
		}
		logger("rpc response", logEntry...)
		if err == nil {
			return nil
		}
		if errors.Is(err, context.DeadlineExceeded) {
			continue
		}
		retryErrs := c.config().retryErrors
		if retryErrs != nil && retryErrs.MatchString(err.Error()) {
			continue
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
			client, err = rpc.DialOptions(ctx, url, rpc.WithWebsocketMessageSizeLimit(c.config().WebsocketMessageSizeLimit))
		} else {
			client, err = rpc.DialOptions(ctx, url, rpc.WithHTTPAuth(node.NewJWTAuth([32]byte(*jwt))), rpc.WithWebsocketMessageSizeLimit(c.config().WebsocketMessageSizeLimit))
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
