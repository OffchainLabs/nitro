package timeboost

import (
	"bytes"
	"context"
	"encoding/csv"
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
	Enable         bool          `koanf:"enable"`
	AccessKey      string        `koanf:"access-key"`
	Bucket         string        `koanf:"bucket"`
	ObjectPrefix   string        `koanf:"object-prefix"`
	Region         string        `koanf:"region"`
	SecretKey      string        `koanf:"secret-key"`
	UploadInterval time.Duration `koanf:"upload-interval"`
	MaxBatchSize   int           `koanf:"max-batch-size"`
	MaxDbRows      int           `koanf:"max-db-rows"`
}

func (c *S3StorageServiceConfig) Validate() error {
	if !c.Enable {
		return nil
	}
	if c.MaxBatchSize < 0 {
		return fmt.Errorf("invalid max-batch-size value for auctioneer's s3-storage config, it should be non-negative, got: %d", c.MaxBatchSize)
	}
	if c.MaxDbRows < 0 {
		return fmt.Errorf("invalid max-db-rows value for auctioneer's s3-storage config, it should be non-negative, got: %d", c.MaxDbRows)
	}
	return nil
}

var DefaultS3StorageServiceConfig = S3StorageServiceConfig{
	Enable:         false,
	UploadInterval: 15 * time.Minute,
	MaxBatchSize:   100000000,
	MaxDbRows:      0, // Disabled by default
}

func S3StorageServiceConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".enable", DefaultS3StorageServiceConfig.Enable, "enable persisting of validated bids to AWS S3 bucket")
	f.String(prefix+".access-key", DefaultS3StorageServiceConfig.AccessKey, "S3 access key")
	f.String(prefix+".bucket", DefaultS3StorageServiceConfig.Bucket, "S3 bucket")
	f.String(prefix+".object-prefix", DefaultS3StorageServiceConfig.ObjectPrefix, "prefix to add to S3 objects")
	f.String(prefix+".region", DefaultS3StorageServiceConfig.Region, "S3 region")
	f.String(prefix+".secret-key", DefaultS3StorageServiceConfig.SecretKey, "S3 secret key")
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
	lastFailedDeleteRound uint64
}

func NewS3StorageService(config *S3StorageServiceConfig, sqlDB *SqliteDatabase) (*S3StorageService, error) {
	client, err := s3client.NewS3FullClient(context.Background(), config.AccessKey, config.SecretKey, config.Region)
	if err != nil {
		return nil, err
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
		ticker := time.NewTicker(s.config.UploadInterval)
		defer ticker.Stop()
		for {
			interval := s.uploadBatches(ctx)
			if ctx.Err() != nil {
				return
			}
			if interval != s.config.UploadInterval { // Indicates error case, so we'll retry sooner than upload-interval
				time.Sleep(interval)
				continue
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}); err != nil {
		log.Error("Failed to launch s3-storage service of auctioneer", "err", err)
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
	if _, err = s.client.Upload(ctx, &putObjectInput); err != nil {
		return err
	}
	return nil
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
			return 5 * time.Second
		}
		s.lastFailedDeleteRound = 0
	}

	bids, round, err := s.sqlDB.GetBids(s.config.MaxDbRows)
	if err != nil {
		log.Error("Error fetching validated bids from sql DB", "round", round, "err", err)
		return 5 * time.Second
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
		return 5 * time.Second
	}
	for index, bid := range bids {
		record := []string{bid.ChainId, bid.Bidder, bid.ExpressLaneController, bid.AuctionContractAddress, fmt.Sprintf("%d", bid.Round), bid.Amount, bid.Signature}
		if err := csvWriter.Write(record); err != nil {
			log.Error("Error writing to csv writer", "err", err)
			return 5 * time.Second
		}
		if s.config.MaxBatchSize != 0 {
			size += csvRecordSize(record)
			if size >= s.config.MaxBatchSize && index < len(bids)-1 && bid.Round != bids[index+1].Round {
				if uploadAndDeleteBids(bids[firstBidId].Round, bid.Round, bids[index+1].Round) != nil {
					return 5 * time.Second
				}
				// Reset csv for next batch
				csvBuffer.Reset()
				if err := csvWriter.Write(header); err != nil {
					log.Error("Error writing to csv writer", "err", err)
					return 5 * time.Second
				}
				size = 0
				firstBidId = index + 1
			}
		}
	}
	if s.config.MaxBatchSize == 0 || size > 0 {
		if uploadAndDeleteBids(bids[firstBidId].Round, bids[len(bids)-1].Round, round) != nil {
			return 5 * time.Second
		}
	}

	if s.lastFailedDeleteRound != 0 {
		return 5 * time.Second
	}

	return s.config.UploadInterval
}
