// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package s3syncer

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"

	"github.com/offchainlabs/nitro/util/s3client"
)

var ErrObjectTooLarge = errors.New("s3 object exceeds max-file-size-mb")

// DataHandler consumes the downloaded object as a stream. size is the number of
// downloaded bytes and digest is the object's ETag, pinned for the whole download.
type DataHandler func(r io.Reader, size int64, digest string) error

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

const bufferedReaderSize = 1 * bytesInMB

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

func (s *Syncer) headAndCheckSize(ctx context.Context) (etag string, err error) {
	headOutput, err := s.client.Client().HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.config.Bucket),
		Key:    aws.String(s.config.ObjectKey),
	})
	if err != nil {
		return "", fmt.Errorf("HeadObject failed for s3://%s/%s: %w", s.config.Bucket, s.config.ObjectKey, err)
	}
	size := aws.ToInt64(headOutput.ContentLength)
	s.objectSizeGauge.Update(size)
	if s.config.MaxFileSizeMB > 0 && size > int64(s.config.MaxFileSizeMB)*bytesInMB {
		return "", fmt.Errorf("%w: %d bytes > %d MB limit (s3://%s/%s)",
			ErrObjectTooLarge, size, s.config.MaxFileSizeMB, s.config.Bucket, s.config.ObjectKey)
	}
	return aws.ToString(headOutput.ETag), nil
}

// CheckAndSync checks if the S3 object has changed (via ETag) and downloads it if so.
func (s *Syncer) CheckAndSync(ctx context.Context) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.client == nil {
		return fmt.Errorf("S3 client not initialized")
	}

	currentETag, err := s.headAndCheckSize(ctx)
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
	return s.downloadAndHandle(ctx, currentETag)
}

// DownloadAndLoad downloads the S3 object and processes it with the data handler.
// This is used for initial load where we need to fetch metadata first.
func (s *Syncer) DownloadAndLoad(ctx context.Context) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.client == nil {
		return fmt.Errorf("S3 client not initialized")
	}

	newETagDigest, err := s.headAndCheckSize(ctx)
	if err != nil {
		return err
	}
	return s.downloadAndHandle(ctx, newETagDigest)
}

// downloadAndHandle downloads the S3 object to a temporary file and streams it
// to the data handler, so the object is never fully buffered in memory.
//
// The disk hop is for speed without increasing the memory footprint.
// Streaming a single GET straight into memory would be capped by S3's
// per-connection throughput, ~85-90 MB/s per AWS's performance guidelines:
// ~3 minutes for a 16 GB object. The SDK downloader instead fetches parts
// over concurrent connections (~900 MB/s at the default concurrency of 10,
// ~18 s for 16 GB). Re-reading the assembled file adds only seconds on an
// NVMe disk (~2-5 GB/s), so concurrent chunked download + disk read beats
// one network stream several times over.
func (s *Syncer) downloadAndHandle(ctx context.Context, etagDigest string) error {
	downloader := manager.NewDownloader(s.client.Client(), func(d *manager.Downloader) {
		d.PartSize = int64(s.config.ChunkSizeMB) * bytesInMB
		d.PartBodyMaxRetries = s.config.MaxRetries
		d.Concurrency = s.config.Concurrency
	})

	f, err := os.CreateTemp(s.config.DownloadDir, "s3sync-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temporary download file: %w", err)
	}
	defer f.Close()
	// Unlink the file up front so a crash mid-download can't strand it on
	// disk; the descriptor keeps it writable and readable until Close, and
	// the kernel reclaims the space then even if the process is killed.
	if err := os.Remove(f.Name()); err != nil {
		log.Warn("failed to unlink temporary download file", "path", f.Name(), "err", err)
	}

	// Download - SDK handles chunking, concurrency, and retry; the parts land at
	// their offsets in the file concurrently. IfMatch pins every part request to
	// the ETag from the HEAD check, so an object replaced mid-download fails the
	// download (to be retried on the next poll) instead of yielding a mix of
	// versions.
	n, err := downloader.Download(ctx, f, &s3.GetObjectInput{
		Bucket:  aws.String(s.config.Bucket),
		Key:     aws.String(s.config.ObjectKey),
		IfMatch: aws.String(etagDigest),
	})
	if err != nil {
		return fmt.Errorf("download failed for s3://%s/%s: %w", s.config.Bucket, s.config.ObjectKey, err)
	}
	// headAndCheckSize already gates on the HEAD-reported size; this guards
	// against a server that ignores IfMatch and serves a larger replacement.
	if s.config.MaxFileSizeMB > 0 && n > int64(s.config.MaxFileSizeMB)*bytesInMB {
		return fmt.Errorf("%w: %d bytes > %d MB limit (s3://%s/%s)",
			ErrObjectTooLarge, n, s.config.MaxFileSizeMB, s.config.Bucket, s.config.ObjectKey)
	}

	return s.applyHandled(etagDigest, bufio.NewReaderSize(io.NewSectionReader(f, 0, n), bufferedReaderSize), n)
}

func (s *Syncer) applyHandled(etagDigest string, r io.Reader, size int64) error {
	if err := s.handleData(r, size, etagDigest); err != nil {
		s.failedETag = etagDigest
		return err
	}
	s.digestETag = etagDigest
	s.failedETag = ""
	return nil
}
