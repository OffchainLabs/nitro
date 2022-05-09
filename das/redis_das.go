// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"bytes"
	"context"
	"encoding/base32"
	"fmt"
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

type RedisDataAvailabilityService struct {
	das         DataAvailabilityService
	redisConfig RedisConfig
	client      redis.UniversalClient
	signerMask  uint64
}

func NewRedisDataAvailabilityService(ctx context.Context, redisConfig RedisConfig, das DataAvailabilityService, signerMask uint64) (*RedisDataAvailabilityService, error) {
	redisOptions, err := redis.ParseURL(redisConfig.RedisUrl)
	if err != nil {
		return nil, err
	}
	return &RedisDataAvailabilityService{
		das:         das,
		redisConfig: redisConfig,
		client:      redis.NewClient(redisOptions),
		signerMask:  signerMask,
	}, nil
}

func (r *RedisDataAvailabilityService) Store(ctx context.Context, message []byte, timeout uint64) (c *arbstate.DataAvailabilityCertificate, err error) {
	c = &arbstate.DataAvailabilityCertificate{}
	copy(c.DataHash[:], crypto.Keccak256(message))

	c.Timeout = timeout
	c.SignersMask = r.signerMask

	fields := serializeSignableFields(*c)
	c.Sig, err = blsSignatures.SignMessage(r.PrivateKey(), fields)
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

func (r *RedisDataAvailabilityService) Retrieve(ctx context.Context, certBytes []byte) ([]byte, error) {
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

func (r *RedisDataAvailabilityService) String() string {
	return fmt.Sprintf("RedisDataAvailabilityService{signersMask:%d}", r.signerMask)
}

func (r *RedisDataAvailabilityService) PrivateKey() blsSignatures.PrivateKey {
	return r.das.PrivateKey()
}
