// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/nitro/blob/master/LICENSE

package das

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/das/dastree"
)

type mockS3Uploader struct {
	mockStorageService StorageService
}

func (m *mockS3Uploader) Upload(ctx context.Context, input *s3.PutObjectInput, opts ...func(*manager.Uploader)) (*manager.UploadOutput, error) {
	buf := new(bytes.Buffer)
	_, err := buf.ReadFrom(input.Body)
	if err != nil {
		return nil, err
	}

	err = m.mockStorageService.Put(ctx, buf.Bytes(), 0)
	return nil, err
}

type mockS3Downloader struct {
	mockStorageService StorageService
}

func (m *mockS3Downloader) Download(ctx context.Context, w io.WriterAt, input *s3.GetObjectInput, options ...func(*manager.Downloader)) (n int64, err error) {
	key, err := DecodeStorageServiceKey(*input.Key)
	if err != nil {
		return 0, err
	}
	res, err := m.mockStorageService.GetByHash(ctx, key)
	if err != nil {
		return 0, err
	}

	ret, err := w.WriteAt(res, 0)
	if err != nil {
		return 0, err
	}
	return int64(ret), nil
}

func NewTestS3StorageService(ctx context.Context, s3Config genericconf.S3Config) (StorageService, error) {
	mockStorageService := NewMemoryBackedStorageService(ctx)
	return &S3StorageService{
		bucket:     s3Config.Bucket,
		uploader:   &mockS3Uploader{mockStorageService},
		downloader: &mockS3Downloader{mockStorageService}}, nil
}

func TestS3StorageService(t *testing.T) {
	ctx := context.Background()
	timeout := uint64(time.Now().Add(time.Hour).Unix())
	s3Service, err := NewTestS3StorageService(ctx, genericconf.DefaultS3Config)
	Require(t, err)

	val1 := []byte("The first value")
	val1CorrectKey := dastree.Hash(val1)
	val2IncorrectKey := dastree.Hash(append(val1, 0))

	_, err = s3Service.GetByHash(ctx, val1CorrectKey)
	if !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}

	err = s3Service.Put(ctx, val1, timeout)
	Require(t, err)

	_, err = s3Service.GetByHash(ctx, val2IncorrectKey)
	if !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}
	val, err := s3Service.GetByHash(ctx, val1CorrectKey)
	Require(t, err)
	if !bytes.Equal(val, val1) {
		t.Fatal(val, val1)
	}
}
