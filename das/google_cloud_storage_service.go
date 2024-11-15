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

type GoogleCloudStorageServiceConfig struct {
	Enable              bool   `koanf:"enable"`
	AccessTokenFile     string `koanf:"access-token-file"`
	Bucket              string `koanf:"bucket"`
	ObjectPrefix        string `koanf:"object-prefix"`
	DiscardAfterTimeout bool   `koanf:"discard-after-timeout"`
}

var DefaultGoogleCloudStorageServiceConfig = GoogleCloudStorageServiceConfig{}

func GoogleCloudConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enable", DefaultGoogleCloudStorageServiceConfig.Enable, "EXPERIMENTAL/unsupported - enable storage/retrieval of sequencer batch data from an Google Cloud Storage bucket")
	f.String(prefix+".access-token-file", DefaultGoogleCloudStorageServiceConfig.AccessTokenFile, "Google Cloud Storage access token")
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
	client, err := buildGoogleCloudStorageClient(config.AccessTokenFile)
	if err != nil {
		return nil, fmt.Errorf("error creating Google Cloud Storage client: %w", err)
	}
	return &GoogleCloudStorageService{
		client:              client,
		bucket:              config.Bucket,
		objectPrefix:        config.ObjectPrefix,
		discardAfterTimeout: config.DiscardAfterTimeout,
	}, nil
}

func buildGoogleCloudStorageClient(accessTokenFile string) (*googlestorage.Client, error) {
	// Note that if the credentials are not specified, the client library will find credentials using ADC(Application Default Credentials)
	// https://cloud.google.com/docs/authentication/provide-credentials-adc.
	if accessTokenFile == "" {
		return googlestorage.NewClient(context.Background())
	}
	return googlestorage.NewClient(context.Background(), option.WithCredentialsFile(accessTokenFile))
}

func (gcs *GoogleCloudStorageService) GetByHash(ctx context.Context, key common.Hash) ([]byte, error) {
	log.Trace("das.GoogleCloudStorageService.GetByHash", "key", pretty.PrettyHash(key), "this", gcs)
	obj := gcs.client.Bucket(gcs.bucket).Object(gcs.objectPrefix + EncodeStorageServiceKey(key))
	reader, err := obj.NewReader(ctx)
	if err != nil {
		log.Error("das.GoogleCloudStorageService.GetByHash", "err", err)
		return nil, err
	}
	buf, err := io.ReadAll(reader)
	if err != nil {
		log.Error("das.GoogleCloudStorageService.GetByHash", "err", err)
	}
	return buf, nil
}

func (gcs *GoogleCloudStorageService) Put(ctx context.Context, value []byte, timeout uint64) error {
	logPut("das.GoogleCloudStorageService.Store", value, timeout, gcs)
	obj := gcs.client.Bucket(gcs.bucket).Object(gcs.objectPrefix + EncodeStorageServiceKey(dastree.Hash(value)))
	w := obj.NewWriter(ctx)
	if gcs.discardAfterTimeout && timeout <= math.MaxInt64 {
		w.Retention = &googlestorage.ObjectRetention{
			Mode:        "Unlocked",
			RetainUntil: time.Unix(int64(timeout), 0),
		}
	}
	if _, err := fmt.Fprintln(w, value); err != nil {
		log.Error("das.GoogleCloudStorageService.Store", "err", err)
		return err
	}
	err := w.Close()
	if err != nil {
		log.Error("das.GoogleCloudStorageService.Store", "err", err)
	}
	return err
}

func (gcs *GoogleCloudStorageService) Sync(ctx context.Context) error {
	return nil
}

func (gcs *GoogleCloudStorageService) Close(ctx context.Context) error {
	return gcs.client.Close()
}

func (gcs *GoogleCloudStorageService) ExpirationPolicy(ctx context.Context) (daprovider.ExpirationPolicy, error) {
	if gcs.discardAfterTimeout {
		return daprovider.DiscardAfterDataTimeout, nil
	}
	return daprovider.KeepForever, nil
}

func (gcs *GoogleCloudStorageService) String() string {
	return fmt.Sprintf("GoogleCloudStorageService(:%s)", gcs.bucket)
}

func (gcs *GoogleCloudStorageService) HealthCheck(ctx context.Context) error {
	bucket := gcs.client.Bucket(gcs.bucket)
	// check if we have bucket permissions
	permissions := []string{
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
