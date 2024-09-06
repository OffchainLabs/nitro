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
	client              *googlestorage.Client
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
		client:              client,
		bucket:              config.Bucket,
		objectPrefix:        config.ObjectPrefix,
		discardAfterTimeout: config.DiscardAfterTimeout,
	}, nil
}

func (gcs *GoogleCloudStorageService) Put(ctx context.Context, value []byte, timeout uint64) error {
	logPut("das.GoogleCloudStorageService.Store", value, timeout, gcs)
	bucket := gcs.client.Bucket(gcs.bucket).Object(gcs.objectPrefix + EncodeStorageServiceKey(dastree.Hash(value)))
	w := bucket.NewWriter(ctx)
	if _, err := fmt.Fprintln(w, hex.EncodeToString(value)); err != nil {
		log.Error("das.GoogleCloudStorageService.Store", "err", err)
		return err
	}
	return w.Close()
}

func (gcs *GoogleCloudStorageService) GetByHash(ctx context.Context, key common.Hash) ([]byte, error) {
	log.Trace("das.GoogleCloudStorageService.GetByHash", "key", pretty.PrettyHash(key), "this", gcs)
	bucket := gcs.client.Bucket(gcs.bucket).Object(gcs.objectPrefix + EncodeStorageServiceKey(key))
	reader, err := bucket.NewReader(ctx)
	if err != nil {
		return nil, err
	}
	buf, err := io.ReadAll(reader)
	if err != nil {
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
	return gcs.client.Close()
}

func (gcs *GoogleCloudStorageService) String() string {
	return fmt.Sprintf("GoogleCloudStorageService(:%s)", gcs.bucket)
}

func (gcs *GoogleCloudStorageService) HealthCheck(ctx context.Context) error {
	bucket := gcs.client.Bucket(gcs.bucket)
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
