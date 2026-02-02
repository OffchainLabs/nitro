// Copyright 2024-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package s3client

import (
	"context"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/spf13/pflag"
)

// Config holds the base S3 connection configuration.
type Config struct {
	AccessKey string `koanf:"access-key"`
	SecretKey string `koanf:"secret-key"`
	Region    string `koanf:"region"`
	Endpoint  string `koanf:"endpoint"`
	Bucket    string `koanf:"bucket"`
}

var DefaultConfig = Config{}

func ConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.String(prefix+".access-key", DefaultConfig.AccessKey, "S3 access key")
	f.String(prefix+".secret-key", DefaultConfig.SecretKey, "S3 secret key")
	f.String(prefix+".region", DefaultConfig.Region, "S3 region")
	f.String(prefix+".endpoint", DefaultConfig.Endpoint, "custom S3 endpoint URL (for MinIO, localstack, or other S3-compatible services)")
	f.String(prefix+".bucket", DefaultConfig.Bucket, "S3 bucket name")
}

func NewS3FullClientFromConfig(ctx context.Context, config *Config) (FullClient, error) {
	return NewS3FullClient(ctx, config.AccessKey, config.SecretKey, config.Region, config.Endpoint)
}

type Uploader interface {
	Upload(ctx context.Context, input *s3.PutObjectInput, opts ...func(*manager.Uploader)) (*manager.UploadOutput, error)
}

type Downloader interface {
	Download(ctx context.Context, w io.WriterAt, input *s3.GetObjectInput, options ...func(*manager.Downloader)) (n int64, err error)
}

type FullClient interface {
	Uploader
	Downloader
	Client() *s3.Client
}

type s3Client struct {
	client     *s3.Client
	uploader   Uploader
	downloader Downloader
}

func NewS3FullClient(ctx context.Context, accessKey, secretKey, region, endpoint string) (FullClient, error) {
	cfg, err := awsConfig.LoadDefaultConfig(ctx, awsConfig.WithRegion(region), func(options *awsConfig.LoadOptions) error {
		// remain backward compatible with accessKey and secretKey credentials provided via cli flags
		if accessKey != "" && secretKey != "" {
			options.Credentials = credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	var client *s3.Client
	if endpoint != "" {
		// Custom endpoint for S3-compatible services like MinIO
		client = s3.NewFromConfig(cfg, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(endpoint)
			o.UsePathStyle = true // Required for MinIO and most S3-compatible services
		})
	} else {
		client = s3.NewFromConfig(cfg)
	}
	return &s3Client{
		client:     client,
		uploader:   manager.NewUploader(client),
		downloader: manager.NewDownloader(client),
	}, nil
}

func (s *s3Client) Client() *s3.Client {
	return s.client
}

func (s *s3Client) Upload(ctx context.Context, input *s3.PutObjectInput, opts ...func(*manager.Uploader)) (*manager.UploadOutput, error) {
	return s.uploader.Upload(ctx, input, opts...)
}

func (s *s3Client) Download(ctx context.Context, w io.WriterAt, input *s3.GetObjectInput, options ...func(*manager.Downloader)) (n int64, err error) {
	return s.downloader.Download(ctx, w, input, options...)
}
