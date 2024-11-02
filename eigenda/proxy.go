package eigenda

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/Layr-Labs/eigenda/api/grpc/disperser"
	"github.com/ethereum/go-ethereum/rlp"
)

var (
	SvcUnavailableErr = fmt.Errorf("eigenda service is unavailable")
)

type EigenDAProxyClient struct {
	client ProxyClient
}

func NewEigenDAProxyClient(rpcUrl string) *EigenDAProxyClient {
	c := New(&Config{
		URL: rpcUrl,
	})
	return &EigenDAProxyClient{client: c}
}

func (c *EigenDAProxyClient) Put(ctx context.Context, data []byte) (*disperser.BlobInfo, error) {
	cert, err := c.client.SetData(ctx, data)
	if err != nil {
		return nil, fmt.Errorf("failed to set data: %w", err)
	}

	var blobInfo disperser.BlobInfo
	err = rlp.DecodeBytes(cert[1:], &blobInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to decode blob info: %w", err)
	}

	return &blobInfo, nil
}

func (c *EigenDAProxyClient) Get(ctx context.Context, blobInfo *DisperserBlobInfo) ([]byte, error) {
	commitment, err := rlp.EncodeToBytes(blobInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to encode blob info: %w", err)
	}

	// TODO: support more strict versioning
	commitWithVersion := append([]byte{0x0}, commitment...)

	data, err := c.client.GetData(ctx, commitWithVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to get data: %w", err)
	}

	return data, nil
}

// TODO: Add support for custom http client option
type Config struct {
	URL string
}

// ProxyClient is an interface for communicating with the EigenDA proxy server
type ProxyClient interface {
	Health() error
	GetData(ctx context.Context, cert []byte) ([]byte, error)
	SetData(ctx context.Context, b []byte) ([]byte, error)
}

// client is the implementation of ProxyClient
type client struct {
	cfg        *Config
	httpClient *http.Client
}

var _ ProxyClient = (*client)(nil)

func New(cfg *Config) ProxyClient {
	return &client{
		cfg,
		http.DefaultClient,
	}
}

// Health indicates if the server is operational; useful for event based awaits
// when integration testing
func (c *client) Health() error {
	url := c.cfg.URL + "/health"
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received bad status code: %d", resp.StatusCode)
	}

	return nil
}

// GetData fetches blob data associated with a DA certificate
func (c *client) GetData(ctx context.Context, comm []byte) ([]byte, error) {
	url := fmt.Sprintf("%s/get/0x%x?commitment_mode=simple", c.cfg.URL, comm)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to construct http request: %w", err)
	}

	req.Header.Set("Content-Type", "application/octet-stream")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received unexpected response code: %d", resp.StatusCode)
	}

	if resp.StatusCode == http.StatusServiceUnavailable {
		return nil, SvcUnavailableErr
	}

	return io.ReadAll(resp.Body)
}

// SetData writes raw byte data to DA and returns the associated certificate
// which should be verified within the proxy
func (c *client) SetData(ctx context.Context, b []byte) ([]byte, error) {
	url := fmt.Sprintf("%s/put/?commitment_mode=simple", c.cfg.URL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to store data: %v", resp.StatusCode)
	}

	b, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if len(b) == 0 {
		return nil, fmt.Errorf("read certificate is empty")
	}

	return b, err
}
