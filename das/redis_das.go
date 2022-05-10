// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"bytes"
	"context"
	"encoding/base32"
	"fmt"
	"os"
	"time"

	"github.com/go-redis/redis/v8"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/blsSignatures"
)

type RedisConfig struct {
	RedisUrl   string        `koanf:"redis-url"`
	Expiration time.Duration `koanf:"redis-expiration"`
}

var DefaultRedisConfig = RedisConfig{
	RedisUrl:   "",
	Expiration: time.Hour,
}

type RedisDAS struct {
	das             DataAvailabilityService
	redisConfig     RedisConfig
	localDiskConfig LocalDiskDASConfig
	privKey         *blsSignatures.PrivateKey
	client          redis.UniversalClient
}

func NewRedisDataAvailabilityService(redisConfig RedisConfig, localDiskConfig LocalDiskDASConfig, das DataAvailabilityService) (*RedisDAS, error) {
	var privKey *blsSignatures.PrivateKey
	var err error
	if len(localDiskConfig.PrivKey) != 0 {
		privKey, err = DecodeBase64BLSPrivateKey([]byte(localDiskConfig.PrivKey))
		if err != nil {
			return nil, fmt.Errorf("'priv-key' was invalid: %w", err)
		}
	} else {
		_, privKey, err = ReadKeysFromFile(localDiskConfig.KeyDir)
		if err != nil {
			if os.IsNotExist(err) {
				if localDiskConfig.AllowGenerateKeys {
					_, privKey, err = GenerateAndStoreKeys(localDiskConfig.KeyDir)
					if err != nil {
						return nil, err
					}
				} else {
					return nil, fmt.Errorf("Required BLS keypair did not exist at %s", localDiskConfig.KeyDir)
				}
			} else {
				return nil, err
			}
		}
	}
	redisOptions, err := redis.ParseURL(redisConfig.RedisUrl)
	if err != nil {
		return nil, err
	}
	return &RedisDAS{
		das:             das,
		redisConfig:     redisConfig,
		localDiskConfig: localDiskConfig,
		privKey:         privKey,
		client:          redis.NewClient(redisOptions),
	}, nil
}

func (r *RedisDAS) Store(ctx context.Context, message []byte, timeout uint64) (c *arbstate.DataAvailabilityCertificate, err error) {
	c = &arbstate.DataAvailabilityCertificate{}
	copy(c.DataHash[:], crypto.Keccak256(message))

	c.Timeout = timeout
	c.SignersMask = 0 // The aggregator decides on the mask for each signer.

	fields := serializeSignableFields(*c)
	c.Sig, err = blsSignatures.SignMessage(*r.privKey, fields)
	if err != nil {
		return nil, err
	}

	path := base32.StdEncoding.EncodeToString(c.DataHash[:])
	err = r.client.Get(ctx, path).Err()
	if err != nil {
		c, err = r.das.Store(ctx, message, timeout)
		if err != nil {
			return nil, err
		}

		r.client.Set(ctx, path, message, r.redisConfig.Expiration)
		return c, err
	}

	return c, err
}

func (r *RedisDAS) Retrieve(ctx context.Context, certBytes []byte) ([]byte, error) {
	cert, err := arbstate.DeserializeDASCertFrom(bytes.NewReader(certBytes))
	if err != nil {
		return nil, err
	}

	path := base32.StdEncoding.EncodeToString(cert.DataHash[:])

	result, err := r.client.Get(ctx, path).Bytes()
	if err != nil {
		result, err = r.das.Retrieve(ctx, certBytes)
		if err != nil {
			return nil, err
		}

		r.client.Set(ctx, path, result, r.redisConfig.Expiration)
		return result, err
	}

	return result, err
}

func (r *RedisDAS) String() string {
	return fmt.Sprintf("RedisDAS{redisConfig:%v, localDiskConfig:%v}", r.redisConfig, r.localDiskConfig)
}
