package das

import (
	googlestorage "cloud.google.com/go/storage"
	"context"
	"errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/das/dastree"
	"testing"
)

type mockGCSClient struct {
}

func (c *mockGCSClient) Bucket(name string) *googlestorage.BucketHandle {
	//TODO implement me
	panic("implement me")
}

func (c *mockGCSClient) Download(ctx context.Context, bucket, objectPrefix string, key common.Hash) ([]byte, error) {
	return nil, ErrNotFound
}

func (c *mockGCSClient) Close(ctx context.Context) error {
	//TODO implement me
	panic("implement me")
}

func (*mockGCSClient) Upload(ctx context.Context, bucket, objectPrefix string, value []byte) error {
	return nil
}

func NewTestGoogleCloudStorageService(ctx context.Context, googleCloudStorageConfig genericconf.GoogleCloudStorageConfig) (StorageService, error) {
	return &GoogleCloudStorageService{
		bucket:       googleCloudStorageConfig.Bucket,
		objectPrefix: googleCloudStorageConfig.ObjectPrefix,
		operator:     &mockGCSClient{},
	}, nil
}

func TestNewGoogleCloudStorageService(t *testing.T) {
	ctx := context.Background()
	//timeout := uint64(time.Now().Add(time.Hour).Unix())
	googleCloudService, err := NewTestGoogleCloudStorageService(ctx, genericconf.DefaultGoogleCloudStorageConfig)
	Require(t, err)

	val1 := []byte("The first value")
	val1CorrectKey := dastree.Hash(val1)
	//val2IncorrectKey := dastree.Hash(append(val1, 0))

	_, err = googleCloudService.GetByHash(ctx, val1CorrectKey)
	if !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}

}
