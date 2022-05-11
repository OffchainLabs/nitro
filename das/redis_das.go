// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/base32"
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/pkg/errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/offchainlabs/nitro/arbstate"
)

const messageKeyPrefix string = "redisDas.msg."
const certKeyPrefix string = "redisDas.cert."

type RedisConfig struct {
	RedisUrl   string        `koanf:"redis-url"`
	Expiration time.Duration `koanf:"redis-expiration"`
	KeyConfig  string        `koanf:"redis-key-config"`
}

var DefaultRedisConfig = RedisConfig{
	RedisUrl:   "",
	Expiration: time.Hour,
}

type RedisDAS struct {
	das         DataAvailabilityService
	redisConfig RedisConfig
	signingKey  common.Hash
	client      redis.UniversalClient
}

func NewRedisDataAvailabilityService(redisConfig RedisConfig, das DataAvailabilityService) (*RedisDAS, error) {
	redisOptions, err := redis.ParseURL(redisConfig.RedisUrl)
	if err != nil {
		return nil, err
	}
	keyIsHex := keyIsHexRegex.Match([]byte(redisConfig.KeyConfig))
	if keyIsHex {
		return nil, errors.New("signing key file contents are not 32 bytes of hex")
	}
	signingKey := common.HexToHash(redisConfig.KeyConfig)
	return &RedisDAS{
		das:         das,
		redisConfig: redisConfig,
		signingKey:  signingKey,
		client:      redis.NewClient(redisOptions),
	}, nil
}

var keyIsHexRegex = regexp.MustCompile("^(0x)?[a-fA-F0-9]{64}$")

func (r *RedisDAS) signMessage(message []byte) []byte {
	hmac := crypto.Keccak256Hash(r.signingKey[:], message)
	return append(hmac[:], message...)
}

func (r *RedisDAS) setMessageAndCert(ctx context.Context, message []byte, c *arbstate.DataAvailabilityCertificate, path string) error {
	r.client.Set(ctx, messageKeyPrefix+path, r.signMessage(message), r.redisConfig.Expiration)
	cBytes, err := json.Marshal(c)
	if err != nil {
		return err
	}
	r.client.Set(ctx, certKeyPrefix+path, r.signMessage(cBytes), r.redisConfig.Expiration)
	return nil
}

func (r *RedisDAS) verifyMessageSignature(data []byte) ([]byte, error) {
	if len(data) < 32 {
		return nil, errors.New("data is too short to contain message signature")
	}
	message := data[32:]
	var haveHmac common.Hash
	copy(haveHmac[:], data[:32])
	expectHmac := crypto.Keccak256Hash(r.signingKey[:], message)
	if subtle.ConstantTimeCompare(expectHmac[:], haveHmac[:]) == 1 {
		return message, nil
	}
	return nil, errors.New("HMAC signature doesn't match expected value(s)")
}

func (r *RedisDAS) getVerifiedData(ctx context.Context, key string) ([]byte, error) {
	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}
	data, err = r.verifyMessageSignature(data)
	if err != nil {
		return nil, err
	}
	return data, err
}

func (r *RedisDAS) Store(ctx context.Context, message []byte, timeout uint64) (c *arbstate.DataAvailabilityCertificate, err error) {
	path := base32.StdEncoding.EncodeToString(crypto.Keccak256(message))
	cBytes, err := r.getVerifiedData(ctx, certKeyPrefix+path)
	if err != nil {
		c, err := r.das.Store(ctx, message, timeout)
		if err != nil {
			return nil, err
		}
		err = r.setMessageAndCert(ctx, message, c, path)
		if err != nil {
			return nil, err
		}
		return c, err
	}

	err = json.Unmarshal(cBytes, c)
	if err != nil {
		return nil, err
	}
	return c, err
}

func (r *RedisDAS) Retrieve(ctx context.Context, certBytes []byte) ([]byte, error) {
	cert, err := arbstate.DeserializeDASCertFrom(bytes.NewReader(certBytes))
	if err != nil {
		return nil, err
	}

	path := base32.StdEncoding.EncodeToString(cert.DataHash[:])

	result, err := r.getVerifiedData(ctx, messageKeyPrefix+path)
	if err != nil {
		result, err = r.das.Retrieve(ctx, certBytes)
		if err != nil {
			return nil, err
		}

		err = r.setMessageAndCert(ctx, result, cert, path)
		if err != nil {
			return nil, err
		}
		return result, err
	}

	return result, err
}

func (r *RedisDAS) String() string {
	return fmt.Sprintf("RedisDAS{redisConfig:%v}", r.redisConfig)
}
