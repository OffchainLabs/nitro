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

	flag "github.com/spf13/pflag"
)

type S3Uploader interface {
	Upload(ctx context.Context, input *s3.PutObjectInput, opts ...func(*manager.Uploader)) (*manager.UploadOutput, error)
}

type S3Downloader interface {
	Download(ctx context.Context, w io.WriterAt, input *s3.GetObjectInput, options ...func(*manager.Downloader)) (n int64, err error)
}

type S3StorageServiceConfig struct {
	Enable    bool   `koanf:"enable"`
	AccessKey string `koanf:"access-key"`
	Bucket    string `koanf:"bucket"`
	Region    string `koanf:"region"`
	SecretKey string `koanf:"secret-key"`
}

var DefaultS3StorageServiceConfig = S3StorageServiceConfig{}

func S3ConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enable", DefaultS3StorageServiceConfig.Enable, "Enable storage/retrieval of sequencer batch data from an AWS S3 bucket")
	f.String(prefix+".access-key", DefaultS3StorageServiceConfig.AccessKey, "S3 access key")
	f.String(prefix+".bucket", DefaultS3StorageServiceConfig.Bucket, "S3 bucket")
	f.String(prefix+".region", DefaultS3StorageServiceConfig.Region, "S3 region")
	f.String(prefix+".secret-key", DefaultS3StorageServiceConfig.SecretKey, "S3 secret key")
}

type S3StorageService struct {
	bucket     string
	uploader   S3Uploader
	downloader S3Downloader
}

func NewS3StorageService(config S3StorageServiceConfig) (StorageService, error) {
	client := s3.New(s3.Options{
		Region:      config.Region,
		Credentials: aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(config.AccessKey, config.SecretKey, "")),
	})
	return &S3StorageService{
		bucket:     config.Bucket,
		uploader:   manager.NewUploader(client),
		downloader: manager.NewDownloader(client)}, nil
}

func (s3s *S3StorageService) GetByHash(ctx context.Context, key []byte) ([]byte, error) {
	buf := manager.NewWriteAtBuffer([]byte{})
	_, err := s3s.downloader.Download(ctx, buf, &s3.GetObjectInput{
		Bucket: aws.String(s3s.bucket),
		Key:    aws.String(base32.StdEncoding.EncodeToString(key)),
	})

	return buf.Bytes(), err
}

func (s3s *S3StorageService) Put(ctx context.Context, value []byte, timeout uint64) error {
	expires := time.Unix(time.Now().Unix()+int64(timeout), 0)
	_, err := s3s.uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket:  aws.String(s3s.bucket),
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
	return fmt.Sprintf("S3StorageService(:%s)", s3s.bucket)
}
