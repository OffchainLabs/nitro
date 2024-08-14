// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package dbutil

import (
	"errors"
	"fmt"
	"os"
	"regexp"

	"github.com/cockroachdb/pebble"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/syndtr/goleveldb/leveldb"
)

func IsErrNotFound(err error) bool {
	return errors.Is(err, leveldb.ErrNotFound) || errors.Is(err, pebble.ErrNotFound) || errors.Is(err, memorydb.ErrMemorydbNotFound)
}

var pebbleNotExistErrorRegex = regexp.MustCompile("pebble: database .* does not exist")

func isPebbleNotExistError(err error) bool {
	return pebbleNotExistErrorRegex.MatchString(err.Error())
}

func isLeveldbNotExistError(err error) bool {
	return os.IsNotExist(err)
}

func IsNotExistError(err error) bool {
	return isLeveldbNotExistError(err) || isPebbleNotExistError(err)
}

var unfinishedConversionCanaryKey = []byte("unfinished-conversion-canary-key")

func PutUnfinishedConversionCanary(db ethdb.KeyValueStore) error {
	return db.Put(unfinishedConversionCanaryKey, []byte{1})
}

func DeleteUnfinishedConversionCanary(db ethdb.KeyValueStore) error {
	return db.Delete(unfinishedConversionCanaryKey)
}

func UnfinishedConversionCheck(db ethdb.KeyValueStore) error {
	unfinished, err := db.Has(unfinishedConversionCanaryKey)
	if err != nil {
		return fmt.Errorf("Failed to check UnfinishedConversionCanaryKey existence: %w", err)
	}
	if unfinished {
		return errors.New("Unfinished conversion canary key detected")
	}
	return nil
}
