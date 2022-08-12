// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package validator

import (
	"bytes"
	"context"
	"crypto/subtle"
	"fmt"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/go-redis/redis/v8"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/pkg/errors"
	flag "github.com/spf13/pflag"
)

type RedisStateTrackerConfig struct {
	Enable                       bool          `koanf:"enable"`
	RedisUrl                     string        `koanf:"redis-url"`
	LockoutDuration              time.Duration `koanf:"lockout-duration"`
	RefreshInterval              time.Duration `koanf:"refresh-interval"`
	SigningKey                   string        `koanf:"signing-key"`
	FallbackVerificationKey      string        `koanf:"fallback-verification-key"`
	DisableSignatureVerification bool          `koanf:"disable-signature-verification"`
}

func RedisStateTrackerConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enable", DefaultRedisStateTrackerConfig.Enable, "enable validator redis state tracker")
	f.String(prefix+".redis-url", DefaultRedisStateTrackerConfig.RedisUrl, "validator state tracker redis url")
	f.Duration(prefix+".lockout-duration", DefaultRedisStateTrackerConfig.LockoutDuration, "validator redis state tracker block validation lockout duration")
	f.Duration(prefix+".refresh-interval", DefaultRedisStateTrackerConfig.RefreshInterval, "validator redis state tracker block validation lockout refresh interval")
	f.String(prefix+".signing-key", DefaultRedisStateTrackerConfig.SigningKey, "validator redis state tracker signing key")
	f.String(prefix+".fallback-verification-key", DefaultRedisStateTrackerConfig.FallbackVerificationKey, "validator redis state tracker fallback verification key")
	f.Bool(prefix+".disable-signature-verification", DefaultRedisStateTrackerConfig.DisableSignatureVerification, "if true, disable signature verification for the validator redis state tracker")
}

var DefaultRedisStateTrackerConfig = RedisStateTrackerConfig{
	Enable:          false,
	LockoutDuration: 5 * time.Minute,
	RefreshInterval: time.Minute,
}

type RedisStateTracker struct {
	config                  RedisStateTrackerConfig
	client                  redis.UniversalClient
	prefix                  string
	signingKey              *[32]byte
	fallbackVerificationKey *[32]byte
}

func NewRedisStateTracker(config RedisStateTrackerConfig, prefix string, genesisBlock *types.Block) (*RedisStateTracker, error) {
	redisOptions, err := redis.ParseURL(config.RedisUrl)
	if err != nil {
		return nil, err
	}
	signingKey, err := arbutil.LoadSigningKey(config.SigningKey)
	if err != nil {
		return nil, err
	}
	if signingKey == nil && !config.DisableSignatureVerification {
		return nil, errors.New("signature verification is enabled but no key is present")
	}
	fallbackVerificationKey, err := arbutil.LoadSigningKey(config.FallbackVerificationKey)
	if err != nil {
		return nil, err
	}
	t := &RedisStateTracker{
		client:                  redis.NewClient(redisOptions),
		config:                  config,
		prefix:                  prefix,
		signingKey:              signingKey,
		fallbackVerificationKey: fallbackVerificationKey,
	}
	// TODO: populate last block validated in Initialize method with genesisBlock if missing
	return t, nil
}

func (t *RedisStateTracker) verifyMessageSignature(key string, value string) ([]byte, error) {
	data := []byte(value)
	if len(data) < 32 {
		return nil, errors.New("data is too short to contain message signature")
	}
	msg := data[32:]
	if t.config.DisableSignatureVerification {
		return msg, nil
	}
	var haveHmac common.Hash
	copy(haveHmac[:], data[:32])

	expectHmac := crypto.Keccak256Hash(t.signingKey[:], []byte(key), msg)
	if subtle.ConstantTimeCompare(expectHmac[:], haveHmac[:]) == 1 {
		return msg, nil
	}

	if t.fallbackVerificationKey != nil {
		expectHmac = crypto.Keccak256Hash(t.fallbackVerificationKey[:], []byte(key), msg)
		if subtle.ConstantTimeCompare(expectHmac[:], haveHmac[:]) == 1 {
			return msg, nil
		}
	}

	if haveHmac == (common.Hash{}) {
		return nil, errors.New("no HMAC signature present but signature verification is enabled")
	} else {
		return nil, errors.New("HMAC signature doesn't match expected value(s)")
	}
}

func (t *RedisStateTracker) signMessage(key string, msg []byte) []byte {
	var hmac [32]byte
	if t.signingKey != nil {
		hmac = crypto.Keccak256Hash(t.signingKey[:], []byte(key), msg)
	}
	return append(hmac[:], msg...)
}

func (t *RedisStateTracker) redisGet(ctx context.Context, client redis.Cmdable, key string) ([]byte, error) {
	res, err := client.Get(ctx, t.prefix+"."+key).Result()
	if err != nil {
		return nil, err
	}
	return t.verifyMessageSignature(key, res)
}

func (t *RedisStateTracker) redisSet(ctx context.Context, client redis.Cmdable, key string, value []byte) error {
	data := t.signMessage(key, value)
	return client.Set(ctx, t.prefix+"."+key, string(data), 0).Err()
}

var lastBlockSeparator = []byte(" \x00")

type lastValidatedMetadata struct {
	blockHash common.Hash
	endPos    GlobalStatePosition
}

const lastBlockValidatedKey = "last-block-validated"

func (t *RedisStateTracker) lastBlockValidatedAndMeta(ctx context.Context, client redis.Cmdable) (uint64, lastValidatedMetadata, error) {
	res, err := t.redisGet(ctx, client, lastBlockValidatedKey)
	if err != nil {
		return 0, lastValidatedMetadata{}, err
	}
	idx := bytes.Index(res, lastBlockSeparator)
	if idx == -1 {
		return 0, lastValidatedMetadata{}, errors.New("last block validated doesn't contain separator")
	}
	blockNumStr := res[:idx]
	blockNum, err := strconv.ParseUint(string(blockNumStr), 10, 64)
	if err != nil {
		return 0, lastValidatedMetadata{}, err
	}
	blockMetaStr := res[(idx + len(lastBlockSeparator)):]
	var blockMeta lastValidatedMetadata
	err = rlp.DecodeBytes(blockMetaStr, &blockMeta)
	if err != nil {
		return 0, lastValidatedMetadata{}, err
	}
	return blockNum, blockMeta, nil
}

func (t *RedisStateTracker) LastBlockValidated(ctx context.Context) (uint64, error) {
	block, _, err := t.lastBlockValidatedAndMeta(ctx, t.client)
	return block, err
}

func (t *RedisStateTracker) LastBlockValidatedAndHash(ctx context.Context) (uint64, common.Hash, error) {
	block, meta, err := t.lastBlockValidatedAndMeta(ctx, t.client)
	return block, meta.blockHash, err
}

func (t *RedisStateTracker) setLastValidated(ctx context.Context, blockNumber uint64, meta lastValidatedMetadata) error {
	firstPart := fmt.Sprintf("%v \x00", blockNumber)
	secondPart, err := rlp.EncodeToBytes(meta)
	if err != nil {
		return err
	}
	val := []byte(firstPart)
	val = append(val, secondPart...)
	return t.redisSet(ctx, t.client, lastBlockValidatedKey, val)
}

type nextValidation struct {
	blockNum uint64
	pos      GlobalStatePosition
}

const nextValidationKey = "next-validation"

func (t *RedisStateTracker) getNextValidation(ctx context.Context, client redis.Cmdable) (uint64, GlobalStatePosition, error) {
	data, err := t.redisGet(ctx, client, nextValidationKey)
	if err != nil {
		return 0, GlobalStatePosition{}, err
	}
	var info nextValidation
	err = rlp.DecodeBytes(data, &info)
	return info.blockNum, info.pos, err
}

func (t *RedisStateTracker) setNextValidation(ctx context.Context, client redis.Cmdable, next nextValidation) error {
	nextValidationData, err := rlp.EncodeToBytes(next)
	if err != nil {
		return err
	}
	return t.redisSet(ctx, client, nextValidationKey, nextValidationData)
}

func (t *RedisStateTracker) GetNextValidation(ctx context.Context) (uint64, GlobalStatePosition, error) {
	return t.getNextValidation(ctx, t.client)
}

func execTestPipe(pipe redis.Pipeliner, ctx context.Context) error {
	cmders, err := pipe.Exec(ctx)
	if err != nil {
		return err
	}
	for _, cmder := range cmders {
		if err := cmder.Err(); err != nil {
			return err
		}
	}
	return nil
}

const statusSubkey = "validation-status"

func (t *RedisStateTracker) getValidationStatusKey(num uint64) string {
	return fmt.Sprintf("%v.%v", statusSubkey, num)
}

func (t *RedisStateTracker) getValidationStatus(ctx context.Context, client redis.Cmdable, num uint64) (validationStatus, error) {
	data, err := t.redisGet(ctx, client, t.getValidationStatusKey(num))
	if err != nil {
		return validationStatus{}, err
	}
	var status validationStatus
	err = rlp.DecodeBytes(data, &status)
	return status, err
}

// TODO: expiry and refresh
func (t *RedisStateTracker) setValidationStatus(ctx context.Context, client redis.Cmdable, num uint64, status validationStatus) error {
	data, err := rlp.EncodeToBytes(status)
	if err != nil {
		return err
	}
	return t.redisSet(ctx, client, t.getValidationStatusKey(num), data)
}

func (t *RedisStateTracker) getPrevHash(ctx context.Context, tx *redis.Tx, nextBlockToValidate uint64) (uint64, common.Hash, error) {
	lastBlockValidated, lastBlockValidatedMeta, err := t.lastBlockValidatedAndMeta(ctx, t.client)
	if err != nil {
		return 0, common.Hash{}, err
	}
	if nextBlockToValidate > lastBlockValidated+1 {
		err = tx.Watch(ctx, t.getValidationStatusKey(nextBlockToValidate-1)).Err()
		if err != nil {
			return 0, common.Hash{}, err
		}
		status, err := t.getValidationStatus(ctx, tx, nextBlockToValidate-1)
		if err != nil {
			return 0, common.Hash{}, err
		}
		return lastBlockValidated, status.blockHash, nil
	} else if nextBlockToValidate == lastBlockValidated+1 {
		return lastBlockValidated, lastBlockValidatedMeta.blockHash, nil
	} else {
		return 0, common.Hash{}, fmt.Errorf("lastBlockValidated is %v but nextBlockToValidate is %v?", lastBlockValidated, nextBlockToValidate)
	}
}

func (t *RedisStateTracker) BeginValidation(ctx context.Context, header *types.Header, startPos GlobalStatePosition, endPos GlobalStatePosition) (bool, error) {
	num := header.Number.Uint64()
	var success bool
	act := func(tx *redis.Tx) error {
		nextBlockToValidate, nextGlobalState, err := t.getNextValidation(ctx, tx)
		if err != nil {
			return err
		}
		if nextBlockToValidate != num || nextGlobalState != startPos {
			return nil
		}
		_, prevHash, err := t.getPrevHash(ctx, tx, nextBlockToValidate)
		if err != nil {
			return err
		}
		if header.ParentHash != prevHash {
			return fmt.Errorf("previous block %v hash is %v but attempting to validate next block with a previous hash of %v", num-1, prevHash, header.ParentHash)
		}
		pipe := tx.TxPipeline()
		status := validationStatus{
			prevHash:    header.ParentHash,
			blockHash:   header.Hash(),
			validated:   false,
			endPosition: endPos,
		}
		err = t.setValidationStatus(ctx, pipe, num, status)
		if err != nil {
			return err
		}
		err = t.setNextValidation(ctx, pipe, nextValidation{
			blockNum: num + 1,
			pos:      endPos,
		})
		if err != nil {
			return err
		}
		success = true
		return execTestPipe(pipe, ctx)
	}
	err := t.client.Watch(ctx, act, t.prefix+"."+lastBlockValidatedKey, t.prefix+"."+nextValidationKey)
	if errors.Is(err, redis.TxFailedErr) {
		return false, nil
	}
	return success, err
}

func (t *RedisStateTracker) tryToAdvanceLastBlockValidated(ctx context.Context) (bool, error) {
	var success bool
	act := func(tx *redis.Tx) error {
		lastBlockValidated, _, err := t.lastBlockValidatedAndMeta(ctx, tx)
		if err != nil {
			return err
		}
		status, err := t.getValidationStatus(ctx, tx, lastBlockValidated+1)
		if err != nil {
			return err
		}
		if !status.validated {
			return nil
		}
		pipe := tx.TxPipeline()
		err = t.setLastValidated(ctx, lastBlockValidated+1, lastValidatedMetadata{
			blockHash: status.blockHash,
			endPos:    status.endPosition,
		})
		if err != nil {
			return err
		}
		success = true
		return execTestPipe(pipe, ctx)
	}
	err := t.client.Watch(ctx, act, t.prefix+"."+lastBlockValidatedKey)
	return success, err
}

func (t *RedisStateTracker) ValidationCompleted(ctx context.Context, initialEntry *validationEntry) (uint64, GlobalStatePosition, error) {
	initialNum := initialEntry.BlockNumber
	act := func(tx *redis.Tx) error {
		status, err := t.getValidationStatus(ctx, tx, initialNum)
		if err != nil {
			return err
		}
		if status.blockHash != initialEntry.BlockHash {
			return fmt.Errorf("completed validation for block %v with hash %v but we have hash %v saved", initialEntry.BlockNumber, initialEntry.BlockHash, status.blockHash)
		}
		status.validated = true
		pipe := tx.TxPipeline()
		err = t.setValidationStatus(ctx, pipe, initialNum, status)
		if err != nil {
			return err
		}
		return execTestPipe(pipe, ctx)
	}
	err := t.client.Watch(ctx, act, t.getValidationStatusKey(initialNum))
	if err != nil {
		return 0, GlobalStatePosition{}, err
	}
	for {
		success, err := t.tryToAdvanceLastBlockValidated(ctx)
		if err != nil {
			if !errors.Is(err, redis.TxFailedErr) {
				log.Error("error updating last block validated in redis", "err", err)
			}
			break
		}
		if !success {
			break
		}
	}
	lastValidated, lastValidatedMeta, err := t.lastBlockValidatedAndMeta(ctx, t.client)
	return lastValidated, lastValidatedMeta.endPos, err
}

func (t *RedisStateTracker) Reorg(ctx context.Context, blockNum uint64, blockHash common.Hash, nextPosition GlobalStatePosition, isValid func(uint64, common.Hash) bool) error {
	act := func(tx *redis.Tx) error {
		nextToValidate, _, err := t.getNextValidation(ctx, tx)
		if err != nil {
			return err
		}
		if nextToValidate <= blockNum+1 {
			return nil
		}
		lastBlockValidated, prevHash, err := t.getPrevHash(ctx, tx, nextToValidate)
		if err != nil {
			return err
		}
		if isValid(nextToValidate-1, prevHash) {
			return nil
		}

		pipe := tx.TxPipeline()

		for i := lastBlockValidated + 1; i < nextToValidate; i++ {
			err = pipe.Del(ctx, t.getValidationStatusKey(i)).Err()
			if err != nil {
				return err
			}
		}
		err = t.setNextValidation(ctx, pipe, nextValidation{
			blockNum: blockNum + 1,
			pos:      nextPosition,
		})
		if err != nil {
			return err
		}

		if lastBlockValidated > blockNum {
			err := t.setLastValidated(ctx, blockNum, lastValidatedMetadata{
				blockHash: blockHash,
				endPos:    nextPosition,
			})
			if err != nil {
				return err
			}
		}

		return execTestPipe(pipe, ctx)
	}
	err := t.client.Watch(ctx, act, t.prefix+"."+lastBlockValidatedKey, t.prefix+"."+nextValidationKey)
	return err
}
