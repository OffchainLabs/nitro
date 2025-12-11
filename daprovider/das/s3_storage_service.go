// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package das

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/daprovider/das/dastree"
	"github.com/offchainlabs/nitro/daprovider/das/dasutil"
	"github.com/offchainlabs/nitro/util/pretty"
	"github.com/offchainlabs/nitro/util/s3client"
)

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

func S3ConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".enable", DefaultS3StorageServiceConfig.Enable, "enable storage/retrieval of sequencer batch data from an AWS S3 bucket")
	f.String(prefix+".access-key", DefaultS3StorageServiceConfig.AccessKey, "S3 access key")
	f.String(prefix+".bucket", DefaultS3StorageServiceConfig.Bucket, "S3 bucket")
	f.String(prefix+".object-prefix", DefaultS3StorageServiceConfig.ObjectPrefix, "prefix to add to S3 objects")
	f.String(prefix+".region", DefaultS3StorageServiceConfig.Region, "S3 region")
	f.String(prefix+".secret-key", DefaultS3StorageServiceConfig.SecretKey, "S3 secret key")
	f.Bool(prefix+".discard-after-timeout", DefaultS3StorageServiceConfig.DiscardAfterTimeout, "this config option is deprecated")
}

type S3StorageService struct {
	client              s3client.FullClient
	bucket              string
	objectPrefix        string
	discardAfterTimeout bool
}

func NewS3StorageService(config S3StorageServiceConfig) (StorageService, error) {
	client, err := s3client.NewS3FullClient(context.Background(), config.AccessKey, config.SecretKey, config.Region)
	if err != nil {
		return nil, err
	}
	if config.DiscardAfterTimeout {
		return nil, errors.New("s3-storage.discard-after-timeout is depreciated and no longer accepted. Expiration for objects uploaded to S3 bucket can be set by adding lifecycle configuration rule to a bucket")
	}
	return &S3StorageService{
		client:              client,
		bucket:              config.Bucket,
		objectPrefix:        config.ObjectPrefix,
		discardAfterTimeout: config.DiscardAfterTimeout,
	}, nil
}

func (s3s *S3StorageService) GetByHash(ctx context.Context, key common.Hash) ([]byte, error) {
	log.Trace("das.S3StorageService.GetByHash", "key", pretty.PrettyHash(key), "this", s3s)

	buf := manager.NewWriteAtBuffer([]byte{})
	_, err := s3s.client.Download(ctx, buf, &s3.GetObjectInput{
		Bucket: aws.String(s3s.bucket),
		Key:    aws.String(s3s.objectPrefix + EncodeStorageServiceKey(key)),
	})
	return buf.Bytes(), err
}

func (s3s *S3StorageService) Put(ctx context.Context, value []byte, _ uint64) error {
	logPut("das.S3StorageService.Store", value, 0, s3s)
	putObjectInput := s3.PutObjectInput{
		Bucket: aws.String(s3s.bucket),
		Key:    aws.String(s3s.objectPrefix + EncodeStorageServiceKey(dastree.Hash(value))),
		Body:   bytes.NewReader(value)}
	_, err := s3s.client.Upload(ctx, &putObjectInput)
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

func (s3s *S3StorageService) ExpirationPolicy(ctx context.Context) (dasutil.ExpirationPolicy, error) {
	// Expiration of data uploaded to S3 bucket is handled directly via LifeCycle configuration of the bucket. Users can choose to add a
	// Lifecycle configuration rule with an expiration action that causes objects with a specific prefix to expire certain days after creation.
	// ref=https://docs.aws.amazon.com/AmazonS3/latest/userguide/lifecycle-expire-general-considerations.html
	return dasutil.KeepForever, nil
}

func (s3s *S3StorageService) String() string {
	return fmt.Sprintf("S3StorageService(:%s)", s3s.bucket)
}

func (s3s *S3StorageService) HealthCheck(ctx context.Context) error {
	_, err := s3s.client.Client().HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(s3s.bucket)})
	return err
}
