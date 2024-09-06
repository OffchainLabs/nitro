package das

import (
	googlestorage "cloud.google.com/go/storage"
	"context"
	"encoding/hex"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/google/go-cmp/cmp"
	"github.com/offchainlabs/nitro/arbstate/daprovider"
	"github.com/offchainlabs/nitro/das/dastree"
	"github.com/offchainlabs/nitro/util/pretty"
	flag "github.com/spf13/pflag"
	"google.golang.org/api/option"
	"io"
	"sort"
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

	if _, err := fmt.Fprintln(w, hex.EncodeToString(value)); err != nil {
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
	Enable              bool   `koanf:"enable"`
	AccessToken         string `koanf:"access-token"`
	Bucket              string `koanf:"bucket"`
	ObjectPrefix        string `koanf:"object-prefix"`
	DiscardAfterTimeout bool   `koanf:"discard-after-timeout"`
}

var DefaultGoogleCloudStorageServiceConfig = GoogleCloudStorageServiceConfig{}

func GoogleCloudConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enable", DefaultGoogleCloudStorageServiceConfig.Enable, "enable storage/retrieval of sequencer batch data from an Google Cloud Storage bucket")
	f.String(prefix+".access-token", DefaultGoogleCloudStorageServiceConfig.AccessToken, "Google Cloud Storage access token")
	f.String(prefix+".bucket", DefaultGoogleCloudStorageServiceConfig.Bucket, "Google Cloud Storage bucket")
	f.String(prefix+".object-prefix", DefaultGoogleCloudStorageServiceConfig.ObjectPrefix, "prefix to add to Google Cloud Storage objects")
	f.Bool(prefix+".discard-after-timeout", DefaultGoogleCloudStorageServiceConfig.DiscardAfterTimeout, "discard data after its expiry timeout")
}

type GoogleCloudStorageService struct {
	operator            GoogleCloudStorageOperator
	bucket              string
	objectPrefix        string
	discardAfterTimeout bool
}

func NewGoogleCloudStorageService(config GoogleCloudStorageServiceConfig) (StorageService, error) {
	client, err := googlestorage.NewClient(context.Background(), option.WithCredentialsJSON([]byte(config.AccessToken)))
	if err != nil {
		return nil, err
	}
	return &GoogleCloudStorageService{
		operator:            &GoogleCloudStorageClient{client: client},
		bucket:              config.Bucket,
		objectPrefix:        config.ObjectPrefix,
		discardAfterTimeout: config.DiscardAfterTimeout,
	}, nil
}

func (gcs *GoogleCloudStorageService) Put(ctx context.Context, value []byte, timeout uint64) error {
	logPut("das.GoogleCloudStorageService.Store", value, timeout, gcs)
	if err := gcs.operator.Upload(ctx, gcs.bucket, gcs.objectPrefix, value); err != nil {
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
	return hex.DecodeString(string(buf))
}

func (gcs *GoogleCloudStorageService) ExpirationPolicy(ctx context.Context) (daprovider.ExpirationPolicy, error) {
	if gcs.discardAfterTimeout {
		return daprovider.DiscardAfterDataTimeout, nil
	}
	return daprovider.KeepForever, nil
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
		return err
	}
	sort.Strings(permissions)
	sort.Strings(perms)
	if !cmp.Equal(perms, permissions) {
		return fmt.Errorf("permissions mismatch (-want +got):\n%s", cmp.Diff(permissions, perms))
	}
	// check if bucket exists (and others)
	_, err = bucket.Attrs(ctx)
	return err
}
