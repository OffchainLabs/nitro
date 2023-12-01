// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"context"
	"fmt"
	"time"

	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/das/dastree"
	"github.com/offchainlabs/nitro/util/pretty"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	googleStorage "cloud.google.com/go/storage"
	flag "github.com/spf13/pflag"
)

type GCSStorageServiceConfig struct {
	Enable                 bool   `koanf:"enable"`
	Bucket                 string `koanf:"bucket"`
	ObjectPrefix           string `koanf:"object-prefix"`
	DiscardAfterTimeout    bool   `koanf:"discard-after-timeout"`
	SyncFromStorageService bool   `koanf:"sync-from-storage-service"`
	SyncToStorageService   bool   `koanf:"sync-to-storage-service"`
}

var DefaultGCSStorageServiceConfig = GCSStorageServiceConfig{}

func GCSConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enable", DefaultGCSStorageServiceConfig.Enable, "enable storage/retrieval of sequencer batch data from an Google cloud storage bucket")
	f.String(prefix+".bucket", DefaultGCSStorageServiceConfig.Bucket, "Google cloud storage bucket")
	f.String(prefix+".object-prefix", DefaultGCSStorageServiceConfig.ObjectPrefix, "prefix to add to GCS objects")
	f.Bool(prefix+".discard-after-timeout", DefaultGCSStorageServiceConfig.DiscardAfterTimeout, "discard data after its expiry timeout")
	f.Bool(prefix+".sync-from-storage-service", DefaultGCSStorageServiceConfig.SyncFromStorageService, "enable gcs to be used as a source for regular sync storage")
	f.Bool(prefix+".sync-to-storage-service", DefaultGCSStorageServiceConfig.SyncToStorageService, "enable gcs to be used as a sink for regular sync storage")
}

type GCSStorageService struct {
	client              *googleStorage.Client
	bucket              string
	objectPrefix        string
	discardAfterTimeout bool
}

func NewGCSStorageService(config GCSStorageServiceConfig) (StorageService, error) {
	// https://cloud.google.com/docs/authentication/provide-credentials-adc
	client, err := googleStorage.NewClient(context.TODO())
	if err != nil {
		return nil, err
	}
	return &GCSStorageService{
		client:              client,
		bucket:              config.Bucket,
		objectPrefix:        config.ObjectPrefix,
		discardAfterTimeout: config.DiscardAfterTimeout,
	}, nil
}

func (gcs *GCSStorageService) GetByHash(ctx context.Context, key common.Hash) ([]byte, error) {
	log.Trace("das.GCSStorageService.GetByHash", "key", pretty.PrettyHash(key), "this", gcs)

	object := gcs.client.Bucket(gcs.bucket).Object(gcs.objectPrefix + EncodeStorageServiceKey(key))
	rc, err := object.NewReader(ctx)

	rv := make([]byte, 0)

	if _, err := rc.Read(rv); err != nil {
		return nil, err
	}

	return rv, err
}

func (gcs *GCSStorageService) Put(ctx context.Context, value []byte, timeout uint64) error {
	logPut("das.GCSStorageService.Store", value, timeout, gcs)

	object := gcs.client.Bucket(gcs.bucket).Object(gcs.objectPrefix + EncodeStorageServiceKey(dastree.Hash(value)))
	object = object.If(googleStorage.Conditions{DoesNotExist: true})

	if !gcs.discardAfterTimeout {
		newCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
		ctx = newCtx
		defer cancel()
	}

	wc := object.NewWriter(ctx)

	_, err := wc.Write(value)

	if err != nil {
		log.Error("das.GCSStorageService.Store", "err", err)
	}

	return err
}

func (gcs *GCSStorageService) putKeyValue(ctx context.Context, key common.Hash, value []byte) error {
	object := gcs.client.Bucket(gcs.bucket).Object(gcs.objectPrefix + EncodeStorageServiceKey(key))
	object = object.If(googleStorage.Conditions{DoesNotExist: true})
	wc := object.NewWriter(ctx)

	_, err := wc.Write(value)

	if err != nil {
		log.Error("das.GCSStorageService.Store", "err", err)
	}

	return err
}

func (gcs *GCSStorageService) Sync(ctx context.Context) error {
	return nil
}

func (gcs *GCSStorageService) Close(ctx context.Context) error {
	return nil
}

func (gcs *GCSStorageService) ExpirationPolicy(ctx context.Context) (arbstate.ExpirationPolicy, error) {
	if gcs.discardAfterTimeout {
		return arbstate.DiscardAfterDataTimeout, nil
	}
	return arbstate.KeepForever, nil
}

func (gcs *GCSStorageService) String() string {
	return fmt.Sprintf("GCSStorageService(:%s)", gcs.bucket)
}

func (gcs *GCSStorageService) HealthCheck(ctx context.Context) error {
	_, err := gcs.client.Bucket(gcs.bucket).Attrs(ctx)
	return err
}
