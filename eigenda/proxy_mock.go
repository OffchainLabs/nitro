package eigenda

import (
	"context"
	"errors"
	"fmt"

	"github.com/Layr-Labs/eigenda/api/grpc/disperser"
)

var (
	mockCert       = []byte{0x01, 0x02, 0x03, 0x04}
	mockBlobData   = []byte("mock data")
	mockBlobInfo   = disperser.BlobInfo{}
	mockHealthErr  = errors.New("service unavailable")
	mockServiceErr = fmt.Errorf("mock error: failed to store data")
)

// MockEigenDA implements the EigenDA interface using the mock client.
type MockEigenDA struct {
	client *MockEigenDAProxyClient
}

// NewMockEigenDA initializes the mock EigenDA instance.
func NewMockEigenDA(fallbackErr bool) *MockEigenDA {
	client := NewMockEigenDAProxyClient(fallbackErr)
	return &MockEigenDA{client: client}
}

// QueryBlob mocks the QueryBlob function.
func (e *MockEigenDA) QueryBlob(ctx context.Context, cert *EigenDABlobInfo, domainFilter string) ([]byte, error) {
	return []byte("mockData"), nil
}

// Store mocks the Store function, returning mock EigenDABlobInfo.
func (e *MockEigenDA) Store(ctx context.Context, data []byte) (*EigenDABlobInfo, error) {
	var blobInfo = &EigenDABlobInfo{}
	cert, err := e.client.Put(ctx, data)
	if err != nil {
		return nil, err
	}

	blobInfo.LoadBlobInfo(cert)

	return blobInfo, nil
}

// Serialize mocks the Serialize function, returning a simple byte slice.
func (e *MockEigenDA) Serialize(blobInfo *EigenDABlobInfo) ([]byte, error) {
	return []byte("mockSerializedData"), nil
}

type MockProxyClient struct {
	ShouldFail      bool // Flag to toggle failure modes
	ShouldReturn503 bool
}

func NewMockProxyClient(failover bool) ProxyClient {
	return &MockProxyClient{
		ShouldFail: failover,
	}
}

func (m *MockProxyClient) Health() error {
	if m.ShouldFail {
		return mockHealthErr
	}
	return nil
}

func (m *MockProxyClient) GetData(ctx context.Context, cert []byte) ([]byte, error) {
	if m.ShouldFail {
		return nil, fmt.Errorf("mock error: failed to get data")
	}
	return mockBlobData, nil
}

func (m *MockProxyClient) SetData(ctx context.Context, data []byte) ([]byte, error) {
	if m.ShouldFail {
		return nil, mockServiceErr
	}
	return mockCert, nil
}

type MockEigenDAProxyClient struct {
	client ProxyClient
}

func NewMockEigenDAProxyClient(shouldFail bool) *MockEigenDAProxyClient {
	return &MockEigenDAProxyClient{client: NewMockProxyClient(shouldFail)}
}

func (c *MockEigenDAProxyClient) Put(ctx context.Context, data []byte) (*disperser.BlobInfo, error) {
	if c.client.(*MockProxyClient).ShouldFail {
		return nil, SvcUnavailableErr
	}
	return &mockBlobInfo, nil
}

func (c *MockEigenDAProxyClient) Get(ctx context.Context, blobInfo *disperser.BlobInfo) ([]byte, error) {
	if c.client.(*MockProxyClient).ShouldFail {
		return nil, fmt.Errorf("mock error: failed to get data")
	}

	if c.client.(*MockProxyClient).ShouldReturn503 {
		return nil, SvcUnavailableErr
	}

	return mockBlobData, nil
}
