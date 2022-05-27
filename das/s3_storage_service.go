// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"bytes"
	"context"
	"encoding/base32"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/offchainlabs/nitro/cmd/genericconf"
)

type S3Uploader interface {
	Upload(ctx context.Context, input *s3.PutObjectInput, opts ...func(*manager.Uploader)) (*manager.UploadOutput, error)
}

type S3Downloader interface {
	Download(ctx context.Context, w io.WriterAt, input *s3.GetObjectInput, options ...func(*manager.Downloader)) (n int64, err error)
}

type S3StorageService struct {
	s3Config   genericconf.S3Config
	uploader   S3Uploader
	downloader S3Downloader
}

func NewS3StorageService(s3Config genericconf.S3Config) (StorageService, error) {
	client := s3.New(s3.Options{
		Region:      s3Config.Region,
		Credentials: aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(s3Config.AccessKey, s3Config.SecretKey, "")),
	})
	return &S3StorageService{
		s3Config:   s3Config,
		uploader:   manager.NewUploader(client),
		downloader: manager.NewDownloader(client)}, nil
}

func (s3s *S3StorageService) GetByHash(ctx context.Context, key []byte) ([]byte, error) {
	buf := manager.NewWriteAtBuffer([]byte{})
	_, err := s3s.downloader.Download(ctx, buf, &s3.GetObjectInput{
		Bucket: aws.String(s3s.s3Config.Bucket),
		Key:    aws.String(base32.StdEncoding.EncodeToString(key)),
	})

	return buf.Bytes(), err
}

func (s3s *S3StorageService) Put(ctx context.Context, value []byte, timeout uint64) error {
	expires := time.Unix(time.Now().Unix()+int64(timeout), 0)
	_, err := s3s.uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket:  aws.String(s3s.s3Config.Bucket),
		Key:     aws.String(base32.StdEncoding.EncodeToString(crypto.Keccak256(value))),
		Body:    bytes.NewReader(value),
		Expires: &expires,
	})
	return err
}

func (s3s *S3StorageService) Sync(ctx context.Context) error {
	return nil
}

func (s3s *S3StorageService) Close(ctx context.Context) error {
	return nil
}

func (s3s *S3StorageService) String() string {
	return fmt.Sprintf("S3StorageService(:%v)", s3s.s3Config)
}
