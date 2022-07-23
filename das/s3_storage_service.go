// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/das/dastree"
	"github.com/offchainlabs/nitro/util/pretty"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	flag "github.com/spf13/pflag"
)

type S3Uploader interface {
	Upload(ctx context.Context, input *s3.PutObjectInput, opts ...func(*manager.Uploader)) (*manager.UploadOutput, error)
}

type S3Downloader interface {
	Download(ctx context.Context, w io.WriterAt, input *s3.GetObjectInput, options ...func(*manager.Downloader)) (n int64, err error)
}

type S3StorageServiceConfig struct {
	Enable              bool   `koanf:"enable"`
	AccessKey           string `koanf:"access-key"`
	Bucket              string `koanf:"bucket"`
	ObjectPrefix        string `koanf:"object-prefix"`
	Region              string `koanf:"region"`
	SecretKey           string `koanf:"secret-key"`
	DiscardAfterTimeout bool   `koanf:"discard-after-timeout"`
}

var DefaultS3StorageServiceConfig = S3StorageServiceConfig{}

func S3ConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enable", DefaultS3StorageServiceConfig.Enable, "enable storage/retrieval of sequencer batch data from an AWS S3 bucket")
	f.String(prefix+".access-key", DefaultS3StorageServiceConfig.AccessKey, "S3 access key")
	f.String(prefix+".bucket", DefaultS3StorageServiceConfig.Bucket, "S3 bucket")
	f.String(prefix+".object-prefix", DefaultS3StorageServiceConfig.ObjectPrefix, "prefix to add to S3 objects")
	f.String(prefix+".region", DefaultS3StorageServiceConfig.Region, "S3 region")
	f.String(prefix+".secret-key", DefaultS3StorageServiceConfig.SecretKey, "S3 secret key")
	f.Bool(prefix+".discard-after-timeout", DefaultS3StorageServiceConfig.DiscardAfterTimeout, "discard data after its expiry timeout")
}

type S3StorageService struct {
	client              *s3.Client
	bucket              string
	objectPrefix        string
	uploader            S3Uploader
	downloader          S3Downloader
	discardAfterTimeout bool
}

func NewS3StorageService(config S3StorageServiceConfig) (StorageService, error) {
	credCache := aws.NewCredentialsCache(
		credentials.NewStaticCredentialsProvider(config.AccessKey, config.SecretKey, ""),
	)
	client := s3.New(s3.Options{
		Region:      config.Region,
		Credentials: credCache,
	})
	return &S3StorageService{
		client:              client,
		bucket:              config.Bucket,
		objectPrefix:        config.ObjectPrefix,
		uploader:            manager.NewUploader(client),
		downloader:          manager.NewDownloader(client),
		discardAfterTimeout: config.DiscardAfterTimeout,
	}, nil
}

func (s3s *S3StorageService) GetByHash(ctx context.Context, key common.Hash) ([]byte, error) {
	log.Trace("das.S3StorageService.GetByHash", "key", pretty.PrettyHash(key), "this", s3s)

	buf := manager.NewWriteAtBuffer([]byte{})
	_, err := s3s.downloader.Download(ctx, buf, &s3.GetObjectInput{
		Bucket: aws.String(s3s.bucket),
		Key:    aws.String(s3s.objectPrefix + EncodeStorageServiceKey(key)),
	})
	return buf.Bytes(), err
}

func (s3s *S3StorageService) Put(ctx context.Context, value []byte, timeout uint64) error {
	logPut("das.S3StorageService.Store", value, timeout, s3s)
	putObjectInput := s3.PutObjectInput{
		Bucket: aws.String(s3s.bucket),
		Key:    aws.String(s3s.objectPrefix + EncodeStorageServiceKey(dastree.Hash(value))),
		Body:   bytes.NewReader(value)}
	if !s3s.discardAfterTimeout {
		expires := time.Unix(int64(timeout), 0)
		putObjectInput.Expires = &expires
	}
	_, err := s3s.uploader.Upload(ctx, &putObjectInput)
	if err != nil {
		log.Error("das.S3StorageService.Store", "err", err)
	}
	return err
}

func (s3s *S3StorageService) Sync(ctx context.Context) error {
	return nil
}

func (s3s *S3StorageService) Close(ctx context.Context) error {
	return nil
}

func (s3s *S3StorageService) ExpirationPolicy(ctx context.Context) (arbstate.ExpirationPolicy, error) {
	if s3s.discardAfterTimeout {
		return arbstate.DiscardAfterDataTimeout, nil
	} else {
		return arbstate.KeepForever, nil
	}
}

func (s3s *S3StorageService) String() string {
	return fmt.Sprintf("S3StorageService(:%s)", s3s.bucket)
}

func (s3s *S3StorageService) HealthCheck(ctx context.Context) error {
	_, err := s3s.client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(s3s.bucket)})
	return err
}
