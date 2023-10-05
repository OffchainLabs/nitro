package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/offchainlabs/nitro/arbutil"
	m "github.com/offchainlabs/nitro/broadcaster/message"
)

func NewHTTPBroadcastClient(c ConfigFetcher) *HTTPBroadcastClient {
	return &HTTPBroadcastClient{c}
}

type HTTPBroadcastClient struct {
	config ConfigFetcher
}

func (c *HTTPBroadcastClient) GetMessages(start, end arbutil.MessageIndex) (*m.BroadcastMessage, error) {
	cfg := c.config()
	url := fmt.Sprintf("%s://%s:%s/?start=%d&end=%d", cfg.Protocol, cfg.Host, cfg.Port, start, end)
	client := http.Client{Timeout: cfg.Timeout}
	res, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error status code returned from %s server: %d - %s", url, res.StatusCode, http.StatusText(res.StatusCode))
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	bm := &m.BroadcastMessage{}
	err = json.Unmarshal(body, bm)
	if err != nil {
		return nil, err
	}

	return bm, nil
}
