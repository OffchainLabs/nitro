package eigenda

import (
	"context"
	"fmt"

	"github.com/Layr-Labs/eigenda-proxy/clients/standard_client"
	"github.com/Layr-Labs/eigenda/api/grpc/disperser"
	"github.com/ethereum/go-ethereum/rlp"
)

type EigenDAProxyClient struct {
	client ProxyClient
}

func NewEigenDAProxyClient(rpcUrl string) *EigenDAProxyClient {
	c := standard_client.New(&standard_client.Config{
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

// ProxyClient is an interface for communicating with the EigenDA proxy server
type ProxyClient interface {
	Health() error
	GetData(ctx context.Context, cert []byte) ([]byte, error)
	SetData(ctx context.Context, b []byte) ([]byte, error)
}
