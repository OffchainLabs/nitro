// Copyright 2024-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package timeboost

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/util/gzip"
	"github.com/offchainlabs/nitro/util/s3client"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

type S3StorageServiceConfig struct {
	s3client.Config `koanf:",squash"`
	Enable          bool          `koanf:"enable"`
	Bucket          string        `koanf:"bucket"`
	ObjectPrefix    string        `koanf:"object-prefix"`
	UploadInterval  time.Duration `koanf:"upload-interval"`
	MaxBatchSize    int           `koanf:"max-batch-size"`
	MaxDbRows       int           `koanf:"max-db-rows"`
}

func (c *S3StorageServiceConfig) Validate() error {
	if !c.Enable {
		return nil
	}
	if c.Bucket == "" {
		return errors.New("s3-storage bucket cannot be empty when enabled")
	}
	if c.Region == "" {
		return errors.New("s3-storage region cannot be empty when enabled")
	}
	if c.UploadInterval <= 0 {
		return fmt.Errorf("s3-storage upload-interval must be positive when enabled, got: %s", c.UploadInterval)
	}
	if c.MaxBatchSize < 0 {
		return fmt.Errorf("s3-storage max-batch-size must be non-negative, got: %d", c.MaxBatchSize)
	}
	if c.MaxDbRows < 0 {
		return fmt.Errorf("s3-storage max-db-rows must be non-negative, got: %d", c.MaxDbRows)
	}
	return nil
}

const s3ErrorRetryInterval = 5 * time.Second

var DefaultS3StorageServiceConfig = S3StorageServiceConfig{
	Enable:         false,
	UploadInterval: 15 * time.Minute,
	MaxBatchSize:   100000000,
	MaxDbRows:      0, // Disabled by default
}

func S3StorageServiceConfigAddOptions(prefix string, f *pflag.FlagSet) {
	s3client.ConfigAddOptions(prefix, f)
	f.Bool(prefix+".enable", DefaultS3StorageServiceConfig.Enable, "enable persisting of validated bids to AWS S3 bucket")
	f.String(prefix+".bucket", DefaultS3StorageServiceConfig.Bucket, "S3 bucket")
	f.String(prefix+".object-prefix", DefaultS3StorageServiceConfig.ObjectPrefix, "prefix to add to S3 objects")
	f.Duration(prefix+".upload-interval", DefaultS3StorageServiceConfig.UploadInterval, "frequency at which batches are uploaded to S3")
	f.Int(prefix+".max-batch-size", DefaultS3StorageServiceConfig.MaxBatchSize, "max size of uncompressed batch in bytes to be uploaded to S3")
	f.Int(prefix+".max-db-rows", DefaultS3StorageServiceConfig.MaxDbRows, "when the sql db is very large, this enables reading of db in chunks instead of all at once which might cause OOM")
}

type S3StorageService struct {
	stopwaiter.StopWaiter
	config                *S3StorageServiceConfig
	client                s3client.FullClient
	sqlDB                 *SqliteDatabase
	bucket                string
	objectPrefix          string
	lastFailedDeleteRound uint64 // only accessed from the uploadBatches LaunchThread
}

func NewS3StorageService(config *S3StorageServiceConfig, sqlDB *SqliteDatabase) (*S3StorageService, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid S3 storage config: %w", err)
	}
	client, err := s3client.NewS3FullClientFromConfig(context.Background(), &config.Config)
	if err != nil {
		return nil, fmt.Errorf("creating S3 client: %w", err)
	}
	return &S3StorageService{
		config:       config,
		client:       client,
		sqlDB:        sqlDB,
		bucket:       config.Bucket,
		objectPrefix: config.ObjectPrefix,
	}, nil
}

func (s *S3StorageService) Start(ctx context.Context) {
	s.StopWaiter.Start(ctx, s)
	if err := s.LaunchThreadSafe(func(ctx context.Context) {
		for {
			interval := s.uploadBatches(ctx)
			if ctx.Err() != nil {
				return
			}
			timer := time.NewTimer(interval)
			select {
			case <-timer.C:
			case <-ctx.Done():
				timer.Stop()
				return
			}
		}
	}); err != nil {
		log.Crit("Failed to launch s3-storage service of auctioneer", "err", err)
	}
}

// Used in padding round numbers to a fixed length for naming the batch being uploaded to s3. <firstRound>-<lastRound>
const fixedRoundStrLen = 7

func (s *S3StorageService) getBatchName(firstRound, lastRound uint64) string {
	padder := "%0" + strconv.Itoa(fixedRoundStrLen) + "d"
	now := time.Now()
	return fmt.Sprintf("%svalidated-timeboost-bids/%d/%02d/%02d/"+padder+"-"+padder+".csv.gzip", s.objectPrefix, now.Year(), now.Month(), now.Day(), firstRound, lastRound)
}
func (s *S3StorageService) uploadBatch(ctx context.Context, batch []byte, firstRound, lastRound uint64) error {
	compressedData, err := gzip.CompressGzip(batch)
	if err != nil {
		return err
	}
	key := s.getBatchName(firstRound, lastRound)
	putObjectInput := s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(compressedData),
	}
	_, err = s.client.Upload(ctx, &putObjectInput)
	return err
}

// downloadBatch is only used for testing purposes
func (s *S3StorageService) downloadBatch(ctx context.Context, key string) ([]byte, error) {
	buf := manager.NewWriteAtBuffer([]byte{})
	if _, err := s.client.Download(ctx, buf, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}); err != nil {
		return nil, err
	}
	return gzip.DecompressGzip(buf.Bytes())
}

func csvRecordSize(record []string) int {
	size := len(record) // comma between fields + newline
	for _, entry := range record {
		size += len(entry)
	}
	return size
}

func (s *S3StorageService) uploadBatches(ctx context.Context) time.Duration {
	// Before doing anything first try to delete the previously uploaded bids that were not successfully erased from the sqlDB
	if s.lastFailedDeleteRound != 0 {
		if err := s.sqlDB.DeleteBids(s.lastFailedDeleteRound); err != nil {
			log.Error("error deleting s3-persisted bids from sql db using lastFailedDeleteRound", "lastFailedDeleteRound", s.lastFailedDeleteRound, "err", err)
			return s3ErrorRetryInterval
		}
		s.lastFailedDeleteRound = 0
	}

	bids, round, err := s.sqlDB.GetBids(s.config.MaxDbRows)
	if err != nil {
		log.Error("Error fetching validated bids from sql DB", "round", round, "err", err)
		return s3ErrorRetryInterval
	}
	// Nothing to persist or a contiguous set of bids wasn't found, so exit early
	if len(bids) == 0 {
		return s.config.UploadInterval
	}

	var csvBuffer bytes.Buffer
	var size int
	var firstBidId int
	csvWriter := csv.NewWriter(&csvBuffer)
	uploadAndDeleteBids := func(firstRound, lastRound, deletRound uint64) error {
		// End current batch when size exceeds MaxBatchSize and the current round ends
		csvWriter.Flush()
		if err := csvWriter.Error(); err != nil {
			log.Error("Error flushing csv writer", "err", err)
			return err
		}
		if err := s.uploadBatch(ctx, csvBuffer.Bytes(), firstRound, lastRound); err != nil {
			log.Error("Error uploading batch to s3", "firstRound", firstRound, "lastRound", lastRound, "err", err)
			return err
		}
		// After successful upload we should go ahead and delete the uploaded bids from DB to prevent duplicate uploads
		// If the delete fails, we track the deleteRound until a future delete succeeds.
		if err := s.sqlDB.DeleteBids(deletRound); err != nil {
			log.Error("error deleting s3-persisted bids from sql db", "round", deletRound, "err", err)
			s.lastFailedDeleteRound = deletRound
		} else {
			// Previously failed deletes don't matter anymore as the recent one (larger round number) succeeded
			s.lastFailedDeleteRound = 0
		}
		return nil
	}

	header := []string{"ChainID", "Bidder", "ExpressLaneController", "AuctionContractAddress", "Round", "Amount", "Signature"}
	if err := csvWriter.Write(header); err != nil {
		log.Error("Error writing to csv writer", "err", err)
		return s3ErrorRetryInterval
	}
	for index, bid := range bids {
		record := []string{bid.ChainId, bid.Bidder, bid.ExpressLaneController, bid.AuctionContractAddress, fmt.Sprintf("%d", bid.Round), bid.Amount, bid.Signature}
		if err := csvWriter.Write(record); err != nil {
			log.Error("Error writing to csv writer", "err", err, "index", index, "round", bid.Round)
			return s3ErrorRetryInterval
		}
		if s.config.MaxBatchSize != 0 {
			size += csvRecordSize(record)
			if size >= s.config.MaxBatchSize && index < len(bids)-1 && bid.Round != bids[index+1].Round {
				if uploadAndDeleteBids(bids[firstBidId].Round, bid.Round, bids[index+1].Round) != nil {
					return s3ErrorRetryInterval
				}
				// Reset csv for next batch
				csvBuffer.Reset()
				if err := csvWriter.Write(header); err != nil {
					log.Error("Error writing to csv writer", "err", err)
					return s3ErrorRetryInterval
				}
				size = 0
				firstBidId = index + 1
			}
		}
	}
	if s.config.MaxBatchSize == 0 || size > 0 {
		if uploadAndDeleteBids(bids[firstBidId].Round, bids[len(bids)-1].Round, round) != nil {
			return s3ErrorRetryInterval
		}
	}

	if s.lastFailedDeleteRound != 0 {
		return s3ErrorRetryInterval
	}

	return s.config.UploadInterval
}
