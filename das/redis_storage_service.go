// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"context"
	"crypto/hmac"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/sha3"

	"github.com/go-redis/redis/v8"
	"github.com/offchainlabs/nitro/arbstate/daprovider"
	"github.com/offchainlabs/nitro/das/dastree"
	"github.com/offchainlabs/nitro/util/pretty"
	"github.com/offchainlabs/nitro/util/redisutil"
	flag "github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

type RedisConfig struct {
	Enable                 bool          `koanf:"enable"`
	Url                    string        `koanf:"url"`
	Expiration             time.Duration `koanf:"expiration"`
	KeyConfig              string        `koanf:"key-config"`
	SyncFromStorageService bool          `koanf:"sync-from-storage-service"`
	SyncToStorageService   bool          `koanf:"sync-to-storage-service"`
}

var DefaultRedisConfig = RedisConfig{
	Url:        "",
	Expiration: time.Hour,
	KeyConfig:  "",
}

func RedisConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enable", DefaultRedisConfig.Enable, "enable Redis caching of sequencer batch data")
	f.String(prefix+".url", DefaultRedisConfig.Url, "Redis url")
	f.Duration(prefix+".expiration", DefaultRedisConfig.Expiration, "Redis expiration")
	f.String(prefix+".key-config", DefaultRedisConfig.KeyConfig, "Redis key config")
	f.Bool(prefix+".sync-from-storage-service", DefaultRedisConfig.SyncFromStorageService, "enable Redis to be used as a source for regular sync storage")
	f.Bool(prefix+".sync-to-storage-service", DefaultRedisConfig.SyncToStorageService, "enable Redis to be used as a sink for regular sync storage")
}

type RedisStorageService struct {
	baseStorageService StorageService
	redisConfig        RedisConfig
	signingKey         common.Hash
	client             redis.UniversalClient
}

func NewRedisStorageService(redisConfig RedisConfig, baseStorageService StorageService) (StorageService, error) {
	redisClient, err := redisutil.RedisClientFromURL(redisConfig.Url)
	if err != nil {
		return nil, err
	}
	signingKey := common.HexToHash(redisConfig.KeyConfig)
	if signingKey == (common.Hash{}) {
		return nil, errors.New("signing key file contents are not 32 bytes of hex")
	}
	return &RedisStorageService{
		baseStorageService: baseStorageService,
		redisConfig:        redisConfig,
		signingKey:         signingKey,
		client:             redisClient,
	}, nil
}

func (rs *RedisStorageService) verifyMessageSignature(data []byte) ([]byte, error) {
	if len(data) < 32 {
		return nil, errors.New("data is too short to contain message signature")
	}
	message := data[:len(data)-32]
	haveHmac := common.BytesToHash(data[len(data)-32:])
	mac := hmac.New(sha3.NewLegacyKeccak256, rs.signingKey[:])
	mac.Write(message)
	expectHmac := mac.Sum(nil)
	if !hmac.Equal(haveHmac[:], expectHmac) {
		return nil, errors.New("HMAC signature doesn't match expected value(s)")
	}
	return message, nil
}

func (rs *RedisStorageService) getVerifiedData(ctx context.Context, key common.Hash) ([]byte, error) {
	data, err := rs.client.Get(ctx, string(key.Bytes())).Bytes()
	if err != nil {
		log.Error("das.RedisStorageService.getVerifiedData", "err", err)
		return nil, err
	}
	data, err = rs.verifyMessageSignature(data)
	if err != nil {
		return nil, err
	}
	return data, err
}

func (rs *RedisStorageService) signMessage(message []byte) []byte {
	mac := hmac.New(sha3.NewLegacyKeccak256, rs.signingKey[:])
	mac.Write(message)
	return mac.Sum(message)
}

func (rs *RedisStorageService) GetByHash(ctx context.Context, key common.Hash) ([]byte, error) {
	log.Trace("das.RedisStorageService.GetByHash", "key", pretty.PrettyHash(key), "this", rs)
	ret, err := rs.getVerifiedData(ctx, key)
	if err != nil {
		ret, err = rs.baseStorageService.GetByHash(ctx, key)
		if err != nil {
			return nil, err
		}

		err = rs.client.Set(ctx, string(key.Bytes()), rs.signMessage(ret), rs.redisConfig.Expiration).Err()
		if err != nil {
			return nil, err
		}
		return ret, err
	}

	return ret, err
}

func (rs *RedisStorageService) Put(ctx context.Context, value []byte, timeout uint64) error {
	logPut("das.RedisStorageService.Store", value, timeout, rs)
	err := rs.baseStorageService.Put(ctx, value, timeout)
	if err != nil {
		return err
	}
	err = rs.client.Set(
		ctx, string(dastree.Hash(value).Bytes()), rs.signMessage(value), rs.redisConfig.Expiration,
	).Err()
	if err != nil {
		log.Error("das.RedisStorageService.Store", "err", err)
	}
	return err
}

func (rs *RedisStorageService) putKeyValue(ctx context.Context, key common.Hash, value []byte) error {
	// Expiration is set to zero here, since we want to keep the index inserted for iterable storage forever.
	err := rs.client.Set(
		ctx, string(key.Bytes()), rs.signMessage(value), 0,
	).Err()
	if err != nil {
		log.Error("das.RedisStorageService.putKeyValue", "err", err)
	}
	return err
}

func (rs *RedisStorageService) Sync(ctx context.Context) error {
	return rs.baseStorageService.Sync(ctx)
}

func (rs *RedisStorageService) Close(ctx context.Context) error {
	err := rs.client.Close()
	if err != nil {
		return err
	}
	return rs.baseStorageService.Close(ctx)
}

func (rs *RedisStorageService) ExpirationPolicy(ctx context.Context) (daprovider.ExpirationPolicy, error) {
	return rs.baseStorageService.ExpirationPolicy(ctx)
}

func (rs *RedisStorageService) String() string {
	return fmt.Sprintf("RedisStorageService(%+v)", rs.redisConfig)
}

func (rs *RedisStorageService) HealthCheck(ctx context.Context) error {
	err := rs.client.Ping(ctx).Err()
	if err != nil {
		return err
	}
	return rs.baseStorageService.HealthCheck(ctx)
}
