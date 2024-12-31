package das

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	googlestorage "cloud.google.com/go/storage"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/das/dastree"
)

type mockGCSClient struct {
	storage map[string][]byte
}

func (c *mockGCSClient) Bucket(name string) *googlestorage.BucketHandle {
	return nil
}

func (c *mockGCSClient) Download(ctx context.Context, bucket, objectPrefix string, key common.Hash) ([]byte, error) {
	value, ok := c.storage[objectPrefix+EncodeStorageServiceKey(key)]
	if !ok {
		return nil, ErrNotFound
	}
	return value, nil
}

func (c *mockGCSClient) Close(ctx context.Context) error {
	return nil
}

func (c *mockGCSClient) Upload(ctx context.Context, bucket, objectPrefix string, value []byte, discardAfterTimeout bool, timeout uint64) error {
	key := objectPrefix + EncodeStorageServiceKey(dastree.Hash(value))
	c.storage[key] = value
	return nil
}

func NewTestGoogleCloudStorageService(ctx context.Context, googleCloudStorageConfig GoogleCloudStorageServiceConfig) (StorageService, error) {
	return &GoogleCloudStorageService{
		bucket:       googleCloudStorageConfig.Bucket,
		objectPrefix: googleCloudStorageConfig.ObjectPrefix,
		operator: &mockGCSClient{
			storage: make(map[string][]byte),
		},
		discardAfterTimeout: true,
	}, nil
}

func TestNewGoogleCloudStorageService(t *testing.T) {
	ctx := context.Background()
	// #nosec G115
	expiry := uint64(time.Now().Add(time.Hour).Unix())
	googleCloudStorageServiceConfig := DefaultGoogleCloudStorageServiceConfig
	googleCloudStorageServiceConfig.Enable = true
	googleCloudService, err := NewTestGoogleCloudStorageService(ctx, googleCloudStorageServiceConfig)
	Require(t, err)

	val1 := []byte("The first value")
	val1CorrectKey := dastree.Hash(val1)
	val2IncorrectKey := dastree.Hash(append(val1, 0))

	_, err = googleCloudService.GetByHash(ctx, val1CorrectKey)
	if !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}

	err = googleCloudService.Put(ctx, val1, expiry)
	Require(t, err)

	_, err = googleCloudService.GetByHash(ctx, val2IncorrectKey)
	if !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}

	val, err := googleCloudService.GetByHash(ctx, val1CorrectKey)
	Require(t, err)
	if !bytes.Equal(val, val1) {
		t.Fatal(val, val1)
	}

}
