package rpcclient

import (
	"context"

	flag "github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/util/signature"
)

type ClientConfig struct {
	URL       string `koanf:"url"`
	JWTSecret string `koanf:"jwtsecret"`
}

var TestClientConfig = ClientConfig{
	URL:       "",
	JWTSecret: "",
}

var DefaultClientConfig = ClientConfig{
	URL:       "ws://127.0.0.1:8549/",
	JWTSecret: "self",
}

func RPCClientAddOptions(prefix string, f *flag.FlagSet) {
	f.String(prefix+".url", DefaultClientConfig.URL, "url of server")
	f.String(prefix+".jwtsecret", DefaultClientConfig.JWTSecret, "path to file with jwtsecret for validation - empty disables jwt, 'self' uses the server's jwt")
}

func CreateRPCClient(ctx context.Context, config *ClientConfig) (*rpc.Client, error) {
	if config.JWTSecret == "" {
		return rpc.DialWebsocket(ctx, config.URL, "")
	}
	jwtHash, err := signature.LoadSigningKey(config.JWTSecret)
	if err != nil {
		return nil, err
	}
	jwt := jwtHash.Bytes()
	return rpc.DialWebsocketJWT(ctx, config.URL, "", jwt)
}
