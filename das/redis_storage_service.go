// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"context"
	"crypto/hmac"
	"fmt"
	"regexp"
	"time"

	"golang.org/x/crypto/sha3"

	"github.com/go-redis/redis/v8"
	"github.com/pkg/errors"
	flag "github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

type RedisConfig struct {
	RedisUrl   string        `koanf:"redis-url"`
	Expiration time.Duration `koanf:"redis-expiration"`
	KeyConfig  string        `koanf:"redis-key-config"`
}

var DefaultRedisConfig = RedisConfig{
	RedisUrl:   "",
	Expiration: time.Hour,
	KeyConfig:  "",
}

func RedisConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.String(prefix+".redis-url", DefaultRedisConfig.RedisUrl, "Redis url")
	f.Duration(prefix+".redis-expiration", DefaultRedisConfig.Expiration, "Redis expiration")
	f.String(prefix+".redis-key-config", DefaultRedisConfig.KeyConfig, "Redis key config")
}

type RedisStorageService struct {
	baseStorageService StorageService
	redisConfig        RedisConfig
	signingKey         common.Hash
	client             redis.UniversalClient
}

var keyIsHexRegex = regexp.MustCompile("^(0x)?[a-fA-F0-9]{64}$")

func NewRedisStorageService(redisConfig RedisConfig, baseStorageService StorageService) (StorageService, error) {
	redisOptions, err := redis.ParseURL(redisConfig.RedisUrl)
	if err != nil {
		return nil, err
	}
	keyIsHex := keyIsHexRegex.Match([]byte(redisConfig.KeyConfig))
	if !keyIsHex {
		return nil, errors.New("signing key file contents are not 32 bytes of hex")
	}
	signingKey := common.HexToHash(redisConfig.KeyConfig)
	return &RedisStorageService{
		baseStorageService: baseStorageService,
		redisConfig:        redisConfig,
		signingKey:         signingKey,
		client:             redis.NewClient(redisOptions),
	}, nil
}

func (rs *RedisStorageService) verifyMessageSignature(data []byte) ([]byte, error) {
	if len(data) < 32 {
		return nil, errors.New("data is too short to contain message signature")
	}
	message := data[:len(data)-32]
	var haveHmac common.Hash
	copy(haveHmac[:], data[len(data)-32:])
	mac := hmac.New(sha3.NewLegacyKeccak256, rs.signingKey[:])
	mac.Write(message)
	expectHmac := mac.Sum(nil)
	if !hmac.Equal(haveHmac[:], expectHmac) {
		return nil, errors.New("HMAC signature doesn't match expected value(s)")
	}
	return message, nil
}

func (rs *RedisStorageService) getVerifiedData(ctx context.Context, key []byte) ([]byte, error) {
	data, err := rs.client.Get(ctx, string(key)).Bytes()
	if err != nil {
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

func (rs *RedisStorageService) GetByHash(ctx context.Context, key []byte) ([]byte, error) {
	ret, err := rs.getVerifiedData(ctx, key)
	if err != nil {
		ret, err = rs.baseStorageService.GetByHash(ctx, key)
		if err != nil {
			return nil, err
		}

		err = rs.client.Set(ctx, string(key), rs.signMessage(ret), rs.redisConfig.Expiration).Err()
		if err != nil {
			return nil, err
		}
		return ret, err
	}

	return ret, err
}

func (rs *RedisStorageService) Put(ctx context.Context, value []byte, timeout uint64) error {
	err := rs.baseStorageService.Put(ctx, value, timeout)
	if err != nil {
		return err
	}
	err = rs.client.Set(ctx, string(crypto.Keccak256(value)), rs.signMessage(value), rs.redisConfig.Expiration).Err()
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

func (rs *RedisStorageService) ExpirationPolicy(ctx context.Context) ExpirationPolicy {
	return rs.baseStorageService.ExpirationPolicy(ctx)
}

func (rs *RedisStorageService) String() string {
	return fmt.Sprintf("RedisStorageService(:%v)", rs.redisConfig)
}
