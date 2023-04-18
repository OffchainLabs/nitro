package rpcclient

import (
	"context"
	"fmt"

	flag "github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/util/signature"
)

type ClientConfig struct {
	URL       string `koanf:"url"`
	JWTSecret string `koanf:"jwtsecret"`
}

var TestClientConfig = ClientConfig{
	URL:       "auto",
	JWTSecret: "",
}

var DefaultClientConfig = ClientConfig{
	URL:       "auto-auth",
	JWTSecret: "",
}

func RPCClientAddOptions(prefix string, f *flag.FlagSet) {
	f.String(prefix+".url", DefaultClientConfig.URL, "url of server, use auto for loopback websocket, auto-auth for loopback with authentication")
	f.String(prefix+".jwtsecret", DefaultClientConfig.JWTSecret, "path to file with jwtsecret for validation - ignored if url is auto or auto-auth")
}

func CreateRPCClient(ctx context.Context, config *ClientConfig, stack *node.Node) (*rpc.Client, error) {
	url := config.URL
	jwtPath := config.JWTSecret
	if url == "auto" {
		url = stack.WSEndpoint()
		jwtPath = ""
	} else if url == "auto-auth" {
		url, jwtPath = stack.AuthEndpoint(true)
	}
	if jwtPath == "" {
		client, err := rpc.DialWebsocket(ctx, url, "")
		if err != nil {
			return nil, fmt.Errorf("%w: url: %s", err, url)
		}
		return client, nil
	}
	jwtHash, err := signature.LoadSigningKey(jwtPath)
	if err != nil {
		return nil, err
	}
	jwt := jwtHash.Bytes()
	return rpc.DialWebsocketJWT(ctx, url, "", jwt)
}
