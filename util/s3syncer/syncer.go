// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package s3syncer

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"

	"github.com/offchainlabs/nitro/util/s3client"
)

var ErrObjectTooLarge = errors.New("s3 object exceeds max-file-size-mb")

// DataHandler processes downloaded data and the associated digest.
type DataHandler func(data []byte, digest string) error

// Syncer handles S3 object syncing with ETag-based change detection.
type Syncer struct {
	client          s3client.FullClient
	config          *Config
	handleData      DataHandler
	objectSizeGauge *metrics.Gauge
	digestETag      string
	failedETag      string
	mutex           sync.Mutex
}

const bytesInMB = 1024 * 1024

func NewSyncer(
	config *Config,
	dataHandler DataHandler,
	objectSizeGauge *metrics.Gauge,
) *Syncer {
	return &Syncer{
		config:          config,
		handleData:      dataHandler,
		objectSizeGauge: objectSizeGauge,
	}
}

func (s *Syncer) Initialize(ctx context.Context) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.client != nil {
		return nil
	}

	client, err := s3client.NewS3FullClientFromConfig(ctx, &s.config.Config)
	if err != nil {
		return fmt.Errorf("failed to create S3 client: %w", err)
	}
	s.client = client
	return nil
}

func (s *Syncer) headAndCheckSize(ctx context.Context) (etag string, size int64, err error) {
	headOutput, err := s.client.Client().HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.config.Bucket),
		Key:    aws.String(s.config.ObjectKey),
	})
	if err != nil {
		return "", 0, fmt.Errorf("HeadObject failed for s3://%s/%s: %w", s.config.Bucket, s.config.ObjectKey, err)
	}
	size = aws.ToInt64(headOutput.ContentLength)
	s.objectSizeGauge.Update(size)
	if s.config.MaxFileSizeMB > 0 && size > int64(s.config.MaxFileSizeMB)*bytesInMB {
		return "", size, fmt.Errorf("%w: %d bytes > %d MB limit (s3://%s/%s)",
			ErrObjectTooLarge, size, s.config.MaxFileSizeMB, s.config.Bucket, s.config.ObjectKey)
	}
	return aws.ToString(headOutput.ETag), size, nil
}

// CheckAndSync checks if the S3 object has changed (via ETag) and downloads it if so.
func (s *Syncer) CheckAndSync(ctx context.Context) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.client == nil {
		return fmt.Errorf("S3 client not initialized")
	}

	currentETag, objectSize, err := s.headAndCheckSize(ctx)
	if err != nil {
		return err
	}

	// Compare with stored digest
	if currentETag == s.digestETag {
		log.Debug("S3 object unchanged", "etag", currentETag, "bucket", s.config.Bucket, "key", s.config.ObjectKey)
		return nil
	}

	if currentETag == s.failedETag {
		log.Warn("S3 object unchanged since last failed load, skipping re-download",
			"etag", currentETag, "bucket", s.config.Bucket, "key", s.config.ObjectKey)
		return nil
	}

	log.Info("S3 object changed, downloading",
		"old_etag", s.digestETag,
		"new_etag", currentETag,
		"bucket", s.config.Bucket,
		"key", s.config.ObjectKey,
	)
	return s.downloadAndHandle(ctx, currentETag, objectSize)
}

// DownloadAndLoad downloads the S3 object and processes it with the data handler.
// This is used for initial load where we need to fetch metadata first.
func (s *Syncer) DownloadAndLoad(ctx context.Context) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.client == nil {
		return fmt.Errorf("S3 client not initialized")
	}

	newETagDigest, objectSize, err := s.headAndCheckSize(ctx)
	if err != nil {
		return err
	}
	return s.downloadAndHandle(ctx, newETagDigest, objectSize)
}

// downloadAndHandle downloads the S3 object to a temp file and calls the data handler.
func (s *Syncer) downloadAndHandle(ctx context.Context, etagDigest string, objectSize int64) error {
	downloader := manager.NewDownloader(s.client.Client(), func(d *manager.Downloader) {
		d.PartSize = int64(s.config.ChunkSizeMB) * bytesInMB
		d.PartBodyMaxRetries = s.config.MaxRetries
		d.Concurrency = s.config.Concurrency
	})

	// let's use an in-memory buffer to avoid file I/O
	buffer := manager.NewWriteAtBuffer(make([]byte, 0, objectSize))

	// Download - SDK handles chunking, concurrency, and retry
	_, err := downloader.Download(ctx, buffer, &s3.GetObjectInput{
		Bucket: aws.String(s.config.Bucket),
		Key:    aws.String(s.config.ObjectKey),
	})
	if err != nil {
		return fmt.Errorf("download failed for s3://%s/%s: %w", s.config.Bucket, s.config.ObjectKey, err)
	}

	return s.applyHandled(etagDigest, buffer.Bytes())
}

func (s *Syncer) applyHandled(etagDigest string, data []byte) error {
	if err := s.handleData(data, etagDigest); err != nil {
		s.failedETag = etagDigest
		return err
	}
	s.digestETag = etagDigest
	s.failedETag = ""
	return nil
}
