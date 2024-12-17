package s3client

import (
	"context"
	"io"

	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

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

func NewS3FullClient(accessKey, secretKey, region string) (FullClient, error) {
	cfg, err := awsConfig.LoadDefaultConfig(context.TODO(), awsConfig.WithRegion(region), func(options *awsConfig.LoadOptions) error {
		// remain backward compatible with accessKey and secretKey credentials provided via cli flags
		if accessKey != "" && secretKey != "" {
			options.Credentials = credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	client := s3.NewFromConfig(cfg)
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
