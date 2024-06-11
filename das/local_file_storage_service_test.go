// Copyright 2024, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"
)

func TestMigrationNoExpiry(t *testing.T) {
	dir := t.TempDir()
	t.Logf("temp dir: %s", dir)
	ctx := context.Background()

	config := LocalFileStorageConfig{
		Enable:       true,
		DataDir:      dir,
		EnableExpiry: false,
		MaxRetention: time.Hour * 24 * 30,
	}
	s, err := NewLocalFileStorageService(config)
	Require(t, err)
	s.enableLegacyLayout = true

	now := uint64(time.Now().Unix())

	err = s.Put(ctx, []byte("a"), now+1)
	Require(t, err)
	err = s.Put(ctx, []byte("b"), now+1)
	Require(t, err)
	err = s.Put(ctx, []byte("c"), now+2)
	Require(t, err)
	err = s.Put(ctx, []byte("d"), now+10)
	Require(t, err)

	err = migrate(&s.legacyLayout, &s.layout)
	Require(t, err)

	migrated := 0
	trIt, err := s.layout.iterateBatches()
	Require(t, err)
	for _, err := trIt.next(); !errors.Is(err, io.EOF); _, err = trIt.next() {
		Require(t, err)
		migrated++
	}
	if migrated != 4 {
		t.Fail()
	}

	//	byTimestampEntries := 0
	trIt, err = s.layout.iterateBatchesByTimestamp(time.Unix(int64(now+10), 0))
	if err == nil {
		t.Fail()
	}
}

func TestMigrationExpiry(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	config := LocalFileStorageConfig{
		Enable:       true,
		DataDir:      dir,
		EnableExpiry: true,
		MaxRetention: time.Hour * 24 * 30,
	}
	s, err := NewLocalFileStorageService(config)
	Require(t, err)
	s.enableLegacyLayout = true

	now := uint64(time.Now().Unix())

	err = s.Put(ctx, []byte("a"), now-expiryDivisor*2)
	Require(t, err)
	err = s.Put(ctx, []byte("b"), now-expiryDivisor)
	Require(t, err)
	err = s.Put(ctx, []byte("c"), now+expiryDivisor)
	Require(t, err)
	err = s.Put(ctx, []byte("d"), now+expiryDivisor)
	Require(t, err)
	err = s.Put(ctx, []byte("e"), now+expiryDivisor*2)
	Require(t, err)

	s.layout.expiryEnabled = true
	err = migrate(&s.legacyLayout, &s.layout)
	Require(t, err)

	migrated := 0
	trIt, err := s.layout.iterateBatches()
	Require(t, err)
	for _, err := trIt.next(); !errors.Is(err, io.EOF); _, err = trIt.next() {
		Require(t, err)
		migrated++
	}
	if migrated != 3 {
		t.Fail()
	}

	countTimestampEntries := func(cutoff, expected uint64) {
		var byTimestampEntries uint64
		trIt, err = s.layout.iterateBatchesByTimestamp(time.Unix(int64(cutoff), 0))
		Require(t, err)
		for batch, err := trIt.next(); !errors.Is(err, io.EOF); batch, err = trIt.next() {
			Require(t, err)
			t.Logf("indexCreated %s", batch)
			byTimestampEntries++
		}
		if byTimestampEntries != expected {
			t.Fail()
		}
	}

	countTimestampEntries(now, 0) // They should have all been filtered out since they're after now
	countTimestampEntries(now+expiryDivisor, 2)
	countTimestampEntries(now+expiryDivisor*2, 3)
}
