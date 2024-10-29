package das

import (
	"context"
	"fmt"
	"io"
	"math"
	"sort"
	"time"

	googlestorage "cloud.google.com/go/storage"
	"github.com/google/go-cmp/cmp"
	flag "github.com/spf13/pflag"
	"google.golang.org/api/option"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbstate/daprovider"
	"github.com/offchainlabs/nitro/das/dastree"
	"github.com/offchainlabs/nitro/util/pretty"
)

type GoogleCloudStorageOperator interface {
	Bucket(name string) *googlestorage.BucketHandle
	Upload(ctx context.Context, bucket, objectPrefix string, value []byte) error
	Download(ctx context.Context, bucket, objectPrefix string, key common.Hash) ([]byte, error)
	Close(ctx context.Context) error
}

type GoogleCloudStorageClient struct {
	client *googlestorage.Client
}

func (g *GoogleCloudStorageClient) Bucket(name string) *googlestorage.BucketHandle {
	return g.client.Bucket(name)
}

func (g *GoogleCloudStorageClient) Upload(ctx context.Context, bucket, objectPrefix string, value []byte) error {
	obj := g.client.Bucket(bucket).Object(objectPrefix + EncodeStorageServiceKey(dastree.Hash(value)))
	w := obj.NewWriter(ctx)

	if _, err := fmt.Fprintln(w, value); err != nil {
		return err
	}
	return w.Close()

}

func (g *GoogleCloudStorageClient) Download(ctx context.Context, bucket, objectPrefix string, key common.Hash) ([]byte, error) {
	obj := g.client.Bucket(bucket).Object(objectPrefix + EncodeStorageServiceKey(key))
	reader, err := obj.NewReader(ctx)
	if err != nil {
		return nil, err
	}
	return io.ReadAll(reader)
}

func (g *GoogleCloudStorageClient) Close(ctx context.Context) error {
	return g.client.Close()
}

type GoogleCloudStorageServiceConfig struct {
	Enable       bool          `koanf:"enable"`
	AccessToken  string        `koanf:"access-token"`
	Bucket       string        `koanf:"bucket"`
	ObjectPrefix string        `koanf:"object-prefix"`
	EnableExpiry bool          `koanf:"enable-expiry"`
	MaxRetention time.Duration `koanf:"max-retention"`
}

var DefaultGoogleCloudStorageServiceConfig = GoogleCloudStorageServiceConfig{}

func GoogleCloudConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enable", DefaultGoogleCloudStorageServiceConfig.Enable, "EXPERIMENTAL/unsupported - enable storage/retrieval of sequencer batch data from an Google Cloud Storage bucket")
	f.String(prefix+".access-token", DefaultGoogleCloudStorageServiceConfig.AccessToken, "Google Cloud Storage access token")
	f.String(prefix+".bucket", DefaultGoogleCloudStorageServiceConfig.Bucket, "Google Cloud Storage bucket")
	f.String(prefix+".object-prefix", DefaultGoogleCloudStorageServiceConfig.ObjectPrefix, "prefix to add to Google Cloud Storage objects")
	f.Bool(prefix+".enable-expiry", DefaultLocalFileStorageConfig.EnableExpiry, "enable expiry of batches")
	f.Duration(prefix+".max-retention", DefaultLocalFileStorageConfig.MaxRetention, "store requests with expiry times farther in the future than max-retention will be rejected")

}

type GoogleCloudStorageService struct {
	operator     GoogleCloudStorageOperator
	bucket       string
	objectPrefix string
	enableExpiry bool
	maxRetention time.Duration
}

func NewGoogleCloudStorageService(config GoogleCloudStorageServiceConfig) (StorageService, error) {
	var client *googlestorage.Client
	var err error
	// Note that if the credentials are not specified, the client library will find credentials using ADC(Application Default Credentials)
	// https://cloud.google.com/docs/authentication/provide-credentials-adc.
	if config.AccessToken == "" {
		client, err = googlestorage.NewClient(context.Background())
	} else {
		client, err = googlestorage.NewClient(context.Background(), option.WithCredentialsJSON([]byte(config.AccessToken)))
	}
	if err != nil {
		return nil, fmt.Errorf("error creating Google Cloud Storage client: %w", err)
	}
	service := &GoogleCloudStorageService{
		operator:     &GoogleCloudStorageClient{client: client},
		bucket:       config.Bucket,
		objectPrefix: config.ObjectPrefix,
		enableExpiry: config.EnableExpiry,
		maxRetention: config.MaxRetention,
	}
	if config.EnableExpiry {
		lifecycleRule := googlestorage.LifecycleRule{
			Action:    googlestorage.LifecycleAction{Type: "Delete"},
			Condition: googlestorage.LifecycleCondition{AgeInDays: int64(config.MaxRetention.Hours() / 24)}, // Objects older than 30 days
		}
		ctx := context.Background()
		bucket := service.operator.Bucket(service.bucket)
		// check if bucket exists (and others), and update expiration policy if enabled
		attrs, err := bucket.Attrs(ctx)
		if err != nil {
			return nil, fmt.Errorf("error getting bucket attributes: %w", err)
		}
		attrs.Lifecycle.Rules = append(attrs.Lifecycle.Rules, lifecycleRule)

		bucketAttrsToUpdate := googlestorage.BucketAttrsToUpdate{
			Lifecycle: &attrs.Lifecycle,
		}
		if _, err := bucket.Update(ctx, bucketAttrsToUpdate); err != nil {
			return nil, fmt.Errorf("failed to update bucket lifecycle: %w", err)
		}
	}
	return service, nil
}

func (gcs *GoogleCloudStorageService) Put(ctx context.Context, data []byte, expiry uint64) error {
	logPut("das.GoogleCloudStorageService.Store", data, expiry, gcs)
	if expiry > math.MaxInt64 {
		return fmt.Errorf("request expiry time (%v) exceeds max int64", expiry)
	}
	// #nosec G115
	expiryTime := time.Unix(int64(expiry), 0)
	currentTimePlusRetention := time.Now().Add(gcs.maxRetention)
	if expiryTime.After(currentTimePlusRetention) {
		return fmt.Errorf("requested expiry time (%v) exceeds current time plus maximum allowed retention period(%v)", expiryTime, currentTimePlusRetention)
	}
	if err := gcs.operator.Upload(ctx, gcs.bucket, gcs.objectPrefix, data); err != nil {
		log.Error("das.GoogleCloudStorageService.Store", "err", err)
		return err
	}
	return nil
}

func (gcs *GoogleCloudStorageService) GetByHash(ctx context.Context, key common.Hash) ([]byte, error) {
	log.Trace("das.GoogleCloudStorageService.GetByHash", "key", pretty.PrettyHash(key), "this", gcs)
	buf, err := gcs.operator.Download(ctx, gcs.bucket, gcs.objectPrefix, key)
	if err != nil {
		log.Error("das.GoogleCloudStorageService.GetByHash", "err", err)
		return nil, err
	}
	return buf, nil
}

func (gcs *GoogleCloudStorageService) ExpirationPolicy(ctx context.Context) (daprovider.ExpirationPolicy, error) {
	if gcs.enableExpiry {
		return daprovider.KeepForever, nil
	}
	return daprovider.DiscardAfterDataTimeout, nil
}

func (gcs *GoogleCloudStorageService) Sync(ctx context.Context) error {
	return nil
}

func (gcs *GoogleCloudStorageService) Close(ctx context.Context) error {
	return gcs.operator.Close(ctx)
}

func (gcs *GoogleCloudStorageService) String() string {
	return fmt.Sprintf("GoogleCloudStorageService(:%s)", gcs.bucket)
}

func (gcs *GoogleCloudStorageService) HealthCheck(ctx context.Context) error {
	bucket := gcs.operator.Bucket(gcs.bucket)
	// check if we have bucket permissions
	permissions := []string{
		"storage.buckets.get",
		"storage.buckets.list",
		"storage.objects.create",
		"storage.objects.delete",
		"storage.objects.list",
		"storage.objects.get",
	}
	perms, err := bucket.IAM().TestPermissions(ctx, permissions)
	if err != nil {
		return fmt.Errorf("could not check permissions: %w", err)
	}
	sort.Strings(permissions)
	sort.Strings(perms)
	if !cmp.Equal(perms, permissions) {
		return fmt.Errorf("permissions mismatch (-want +got):\n%s", cmp.Diff(permissions, perms))
	}

	return nil
}
