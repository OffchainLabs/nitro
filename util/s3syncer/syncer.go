// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package s3syncer

import (
	"context"
	"fmt"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/util/s3client"
)

// DataHandler processes downloaded data and the associated digest.
type DataHandler func(data []byte, digest string) error

// Syncer handles S3 object syncing with ETag-based change detection.
type Syncer struct {
	client     s3client.FullClient
	config     *Config
	handleData DataHandler
	digestETag string
	mutex      sync.Mutex
}

const bytesInMB = 1024 * 1024

// NewSyncer creates a new S3 syncer with the given callbacks.
func NewSyncer(
	ctx context.Context,
	config *Config,
	dataHandler DataHandler,
) (*Syncer, error) {
	s := &Syncer{
		config:     config,
		handleData: dataHandler,
	}

	client, err := s3client.NewS3FullClientFromConfig(ctx, &config.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 client: %w", err)
	}
	s.client = client

	return s, nil
}

// CheckAndSync checks if the S3 object has changed (via ETag) and downloads it if so.
func (s *Syncer) CheckAndSync(ctx context.Context) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	headOutput, err := s.client.Client().HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.config.Bucket),
		Key:    aws.String(s.config.ObjectKey),
	})
	if err != nil {
		return fmt.Errorf("HeadObject failed for s3://%s/%s: %w", s.config.Bucket, s.config.ObjectKey, err)
	}

	currentETag := aws.ToString(headOutput.ETag)

	// Compare with stored digest
	if currentETag == s.digestETag {
		log.Debug("S3 object unchanged", "etag", currentETag, "bucket", s.config.Bucket, "key", s.config.ObjectKey)
		return nil
	}

	log.Info("S3 object changed, downloading",
		"old_etag", s.digestETag,
		"new_etag", currentETag,
		"bucket", s.config.Bucket,
		"key", s.config.ObjectKey,
	)
	objectSize := aws.ToInt64(headOutput.ContentLength)
	return s.downloadAndHandle(ctx, currentETag, objectSize)
}

// DownloadAndLoad downloads the S3 object and processes it with the data handler.
// This is used for initial load where we need to fetch metadata first.
func (s *Syncer) DownloadAndLoad(ctx context.Context) error {
	headOutput, err := s.client.Client().HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.config.Bucket),
		Key:    aws.String(s.config.ObjectKey),
	})
	if err != nil {
		return fmt.Errorf("HeadObject failed for s3://%s/%s: %w", s.config.Bucket, s.config.ObjectKey, err)
	}

	newETagDigest := aws.ToString(headOutput.ETag)
	objectSize := aws.ToInt64(headOutput.ContentLength)
	err = s.downloadAndHandle(ctx, newETagDigest, objectSize)
	return err
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

	err = s.handleData(buffer.Bytes(), etagDigest)
	if err == nil {
		s.digestETag = etagDigest
	}
	return err
}
