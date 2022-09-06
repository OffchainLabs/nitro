package arbrpc

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/ethereum/go-ethereum/rpc"
)

func DialTransport(ctx context.Context, rawUrl string, transport *http.Transport) (*rpc.Client, error) {
	u, err := url.Parse(rawUrl)
	if err != nil {
		return nil, err
	}

	var rpcClient *rpc.Client
	switch u.Scheme {
	case "http", "https":
		client := &http.Client{
			Transport: transport,
		}
		rpcClient, err = rpc.DialHTTPWithClient(rawUrl, client)
	case "ws", "wss":
		rpcClient, err = rpc.DialWebsocket(ctx, rawUrl, "")
	default:
		return nil, fmt.Errorf("no known transport for scheme %q in URL %s", u.Scheme, rawUrl)
	}
	if err != nil {
		return nil, err
	}
	return rpcClient, nil
}
