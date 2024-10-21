package timeboost

import (
	"bytes"
	"context"
	"encoding/csv"

	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/util"
	"github.com/offchainlabs/nitro/util/s3client"
	"github.com/offchainlabs/nitro/util/stopwaiter"
	"github.com/spf13/pflag"
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
}

func (c *S3StorageServiceConfig) Validate() error {
	if !c.Enable {
		return nil
	}
	if c.MaxBatchSize < 0 {
		return fmt.Errorf("invalid max-batch-size value for auctioneer's s3-storage config, it should be non-negative, got: %d", c.MaxBatchSize)
	}
	return nil
}

var DefaultS3StorageServiceConfig = S3StorageServiceConfig{
	Enable:         false,
	UploadInterval: time.Minute, // is this the right default value?
	MaxBatchSize:   100000000,   // is this the right default value?
}

func S3StorageServiceConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".enable", DefaultS3StorageServiceConfig.Enable, "enable persisting of valdiated bids to AWS S3 bucket")
	f.String(prefix+".access-key", DefaultS3StorageServiceConfig.AccessKey, "S3 access key")
	f.String(prefix+".bucket", DefaultS3StorageServiceConfig.Bucket, "S3 bucket")
	f.String(prefix+".object-prefix", DefaultS3StorageServiceConfig.ObjectPrefix, "prefix to add to S3 objects")
	f.String(prefix+".region", DefaultS3StorageServiceConfig.Region, "S3 region")
	f.String(prefix+".secret-key", DefaultS3StorageServiceConfig.SecretKey, "S3 secret key")
	f.Duration(prefix+".upload-interval", DefaultS3StorageServiceConfig.UploadInterval, "frequency at which batches are uploaded to S3")
	f.Int(prefix+".max-batch-size", DefaultS3StorageServiceConfig.MaxBatchSize, "max size of uncompressed batch in bytes to be uploaded to S3")
}

type S3StorageService struct {
	stopwaiter.StopWaiter
	config       *S3StorageServiceConfig
	client       s3client.FullClient
	sqlDB        *SqliteDatabase
	bucket       string
	objectPrefix string
}

func NewS3StorageService(config *S3StorageServiceConfig, sqlDB *SqliteDatabase) (*S3StorageService, error) {
	client, err := s3client.NewS3FullClient(config.AccessKey, config.SecretKey, config.Region)
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
	s.CallIteratively(s.uploadBatches)
}

func (s *S3StorageService) uploadBatch(ctx context.Context, batch []byte, fistRound uint64) error {
	compressedData, err := util.CompressGzip(batch)
	if err != nil {
		return err
	}
	now := time.Now()
	key := fmt.Sprintf("%svalidated-timeboost-bids/%d/%02d/%02d/%d.csv.gzip", s.objectPrefix, now.Year(), now.Month(), now.Day(), fistRound)
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
	return util.DecompressGzip(buf.Bytes())
}

func csvRecordSize(record []string) int {
	size := len(record) // comma between fields + newline
	for _, entry := range record {
		size += len(entry)
	}
	return size
}

func (s *S3StorageService) uploadBatches(ctx context.Context) time.Duration {
	round, err := s.sqlDB.GetMaxRoundFromBids()
	if err != nil {
		log.Error("Error finding max round from validated bids", "err", err)
		return 0
	}
	bids, err := s.sqlDB.GetBidsTillRound(round)
	if err != nil {
		log.Error("Error fetching validated bids from sql DB", "round", round, "err", err)
		return 0
	}
	var csvBuffer bytes.Buffer
	var size int
	var firstBidId int
	csvWriter := csv.NewWriter(&csvBuffer)
	header := []string{"ChainID", "Bidder", "ExpressLaneController", "AuctionContractAddress", "Round", "Amount", "Signature"}
	if err := csvWriter.Write(header); err != nil {
		log.Error("Error writing to csv writer", "err", err)
		return 0
	}
	for index, bid := range bids {
		record := []string{bid.ChainId, bid.Bidder, bid.ExpressLaneController, bid.AuctionContractAddress, fmt.Sprintf("%d", bid.Round), bid.Amount, bid.Signature}
		if err := csvWriter.Write(record); err != nil {
			log.Error("Error writing to csv writer", "err", err)
			return 0
		}
		if s.config.MaxBatchSize != 0 {
			size += csvRecordSize(record)
			if size >= s.config.MaxBatchSize && index < len(bids)-1 && bid.Round != bids[index+1].Round {
				// End current batch when size exceeds MaxBatchSize and the current round ends
				csvWriter.Flush()
				if err := csvWriter.Error(); err != nil {
					log.Error("Error flushing csv writer", "err", err)
					return 0
				}
				if err := s.uploadBatch(ctx, csvBuffer.Bytes(), bids[firstBidId].Round); err != nil {
					log.Error("Error uploading batch to s3", "firstRound", bids[firstBidId].Round, "err", err)
					return 0
				}
				// Reset csv for next batch
				csvBuffer.Reset()
				if err := csvWriter.Write(header); err != nil {
					log.Error("Error writing to csv writer", "err", err)
					return 0
				}
				size = 0
				firstBidId = index + 1
			}
		}
	}
	if (s.config.MaxBatchSize == 0 && len(bids) > 0) || size > 0 {
		csvWriter.Flush()
		if err := csvWriter.Error(); err != nil {
			log.Error("Error flushing csv writer", "err", err)
			return 0
		}
		if err := s.uploadBatch(ctx, csvBuffer.Bytes(), bids[firstBidId].Round); err != nil {
			log.Error("Error uploading batch to s3", "firstRound", bids[firstBidId].Round, "err", err)
			return 0
		}
	}
	if err := s.sqlDB.DeleteBids(round); err != nil {
		return 0
	}
	return s.config.UploadInterval
}
