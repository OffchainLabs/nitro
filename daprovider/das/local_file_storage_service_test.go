// Copyright 2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package das

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/offchainlabs/nitro/daprovider/das/dastree"
)

func getByHashAndCheck(t *testing.T, s *LocalFileStorageService, xs ...string) {
	t.Helper()
	ctx := context.Background()

	for _, x := range xs {
		actual, err := s.GetByHash(ctx, dastree.Hash([]byte(x)))
		Require(t, err)
		if !bytes.Equal([]byte(x), actual) {
			Fail(t, "unexpected result")
		}
	}
}

func countEntries(t *testing.T, layout *trieLayout, expected int) {
	t.Helper()

	count := 0
	trIt, err := layout.iterateBatches()
	Require(t, err)
	for _, err := trIt.next(); !errors.Is(err, io.EOF); _, err = trIt.next() {
		Require(t, err)
		count++
	}
	if count != expected {
		Fail(t, "unexpected number of batches", "expected", expected, "was", count)
	}
}

func countTimestampEntries(t *testing.T, layout *trieLayout, cutoff time.Time, expected int) {
	t.Helper()
	var count int
	trIt, err := layout.iterateBatchesByTimestamp(cutoff)
	Require(t, err)
	for _, err := trIt.next(); !errors.Is(err, io.EOF); _, err = trIt.next() {
		Require(t, err)
		count++
	}
	if count != expected {
		Fail(t, "unexpected count of entries when iterating by timestamp", "expected", expected, "was", count)
	}
}

func pruneCountRemaining(t *testing.T, layout *trieLayout, pruneTil time.Time, expected int) {
	t.Helper()
	err := layout.prune(pruneTil)
	Require(t, err)

	countEntries(t, layout, expected)
}

func TestMigrationNoExpiry(t *testing.T) {
	dir := t.TempDir()
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

	// #nosec G115
	now := uint64(time.Now().Unix())

	err = s.Put(ctx, []byte("a"), now+1)
	Require(t, err)
	err = s.Put(ctx, []byte("b"), now+1)
	Require(t, err)
	err = s.Put(ctx, []byte("c"), now+2)
	Require(t, err)
	err = s.Put(ctx, []byte("d"), now+10)
	Require(t, err)

	getByHashAndCheck(t, s, "a", "b", "c", "d")

	err = migrate(&s.legacyLayout, &s.layout)
	Require(t, err)
	s.enableLegacyLayout = false

	countEntries(t, &s.layout, 4)
	getByHashAndCheck(t, s, "a", "b", "c", "d")

	// Can still iterate by timestamp even if expiry disabled
	// #nosec G115
	countTimestampEntries(t, &s.layout, time.Unix(int64(now+11), 0), 4)

}

func TestMigrationExpiry(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	config := LocalFileStorageConfig{
		Enable:       true,
		DataDir:      dir,
		EnableExpiry: true,
		MaxRetention: time.Hour * 10,
	}
	s, err := NewLocalFileStorageService(config)
	Require(t, err)
	s.enableLegacyLayout = true

	now := time.Now()

	// Use increments of expiry divisor in order to span multiple by-expiry-timestamp dirs
	// #nosec G115
	err = s.Put(ctx, []byte("a"), uint64(now.Add(-2*time.Second*expiryDivisor).Unix()))
	Require(t, err)
	// #nosec G115
	err = s.Put(ctx, []byte("b"), uint64(now.Add(-1*time.Second*expiryDivisor).Unix()))
	Require(t, err)
	// #nosec G115
	err = s.Put(ctx, []byte("c"), uint64(now.Add(time.Second*expiryDivisor).Unix()))
	Require(t, err)
	// #nosec G115
	err = s.Put(ctx, []byte("d"), uint64(now.Add(time.Second*expiryDivisor).Unix()))
	Require(t, err)
	// #nosec G115
	err = s.Put(ctx, []byte("e"), uint64(now.Add(2*time.Second*expiryDivisor).Unix()))
	Require(t, err)

	getByHashAndCheck(t, s, "a", "b", "c", "d", "e")

	err = migrate(&s.legacyLayout, &s.layout)
	Require(t, err)
	s.enableLegacyLayout = false

	countEntries(t, &s.layout, 3)
	getByHashAndCheck(t, s, "c", "d", "e")

	afterNow := now.Add(time.Second)
	countTimestampEntries(t, &s.layout, afterNow, 0) // They should have all been filtered out since they're after now
	countTimestampEntries(t, &s.layout, afterNow.Add(time.Second*expiryDivisor), 2)
	countTimestampEntries(t, &s.layout, afterNow.Add(2*time.Second*expiryDivisor), 3)

	pruneCountRemaining(t, &s.layout, afterNow, 3)
	getByHashAndCheck(t, s, "c", "d", "e")

	pruneCountRemaining(t, &s.layout, afterNow.Add(time.Second*expiryDivisor), 1)
	getByHashAndCheck(t, s, "e")

	pruneCountRemaining(t, &s.layout, afterNow.Add(2*time.Second*expiryDivisor), 0)
}

func TestExpiryDuplicates(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	config := LocalFileStorageConfig{
		Enable:       true,
		DataDir:      dir,
		EnableExpiry: true,
		MaxRetention: time.Hour * 10,
	}
	s, err := NewLocalFileStorageService(config)
	Require(t, err)

	now := time.Now()

	// Use increments of expiry divisor in order to span multiple by-expiry-timestamp dirs
	// #nosec G115
	err = s.Put(ctx, []byte("a"), uint64(now.Add(-2*time.Second*expiryDivisor).Unix()))
	Require(t, err)
	// #nosec G115
	err = s.Put(ctx, []byte("a"), uint64(now.Add(-1*time.Second*expiryDivisor).Unix()))
	Require(t, err)
	// #nosec G115
	err = s.Put(ctx, []byte("a"), uint64(now.Add(time.Second*expiryDivisor).Unix()))
	Require(t, err)
	// #nosec G115
	err = s.Put(ctx, []byte("d"), uint64(now.Add(time.Second*expiryDivisor).Unix()))
	Require(t, err)
	// #nosec G115
	err = s.Put(ctx, []byte("e"), uint64(now.Add(2*time.Second*expiryDivisor).Unix()))
	Require(t, err)
	// #nosec G115
	err = s.Put(ctx, []byte("f"), uint64(now.Add(3*time.Second*expiryDivisor).Unix()))
	Require(t, err)
	// Put the same entry and expiry again, should have no effect
	// #nosec G115
	err = s.Put(ctx, []byte("f"), uint64(now.Add(3*time.Second*expiryDivisor).Unix()))
	Require(t, err)

	afterNow := now.Add(time.Second)
	// "a" is duplicated
	countEntries(t, &s.layout, 4)
	// There should be a timestamp entry for each time "a" was added
	countTimestampEntries(t, &s.layout, afterNow.Add(1000*time.Hour), 6)

	// We've expired the first "a", but there are still 2 other timestamp entries for it
	pruneCountRemaining(t, &s.layout, afterNow.Add(-2*time.Second*expiryDivisor), 4)
	countTimestampEntries(t, &s.layout, afterNow.Add(1000*time.Hour), 5)

	// We've expired the second "a", but there is still 1 other timestamp entry for it
	pruneCountRemaining(t, &s.layout, afterNow.Add(-1*time.Second*expiryDivisor), 4)
	countTimestampEntries(t, &s.layout, afterNow.Add(1000*time.Hour), 4)

	// We've expired the third "a", and also "d"
	pruneCountRemaining(t, &s.layout, afterNow.Add(time.Second*expiryDivisor), 2)
	countTimestampEntries(t, &s.layout, afterNow.Add(1000*time.Hour), 2)

	// We've expired the "e"
	pruneCountRemaining(t, &s.layout, afterNow.Add(2*time.Second*expiryDivisor), 1)
	countTimestampEntries(t, &s.layout, afterNow.Add(1000*time.Hour), 1)

	// We've expired the "f"
	pruneCountRemaining(t, &s.layout, afterNow.Add(3*time.Second*expiryDivisor), 0)
	countTimestampEntries(t, &s.layout, afterNow.Add(1000*time.Hour), 0)
}

// --- Tests for readDirNamesFiltered helper ---

func TestReadDirNamesFiltered_NonExistentPath(t *testing.T) {
	tmp := t.TempDir()
	missing := filepath.Join(tmp, "does-not-exist")

	names, err := readDirNamesFiltered(missing, true)
	if err != nil {
		t.Fatalf("expected nil error for non-existent path, got %v", err)
	}
	if len(names) != 0 {
		t.Fatalf("expected empty list for non-existent path, got %v", names)
	}
}

func TestReadDirNamesFiltered_NonDirectory(t *testing.T) {
	tmp := t.TempDir()
	filePath := filepath.Join(tmp, "file.txt")
	if err := os.WriteFile(filePath, []byte("content"), 0o600); err != nil {
		t.Fatalf("failed creating file: %v", err)
	}

	names, err := readDirNamesFiltered(filePath, true)
	if err != nil {
		t.Fatalf("expected nil error for non-directory path, got %v", err)
	}
	if len(names) != 0 {
		t.Fatalf("expected empty list for non-directory path, got %v", names)
	}
}

func TestReadDirNamesFiltered_FilterDirsAndFiles(t *testing.T) {
	tmp := t.TempDir()
	// Create entries: dirA (dir), fileB (file), dirC (dir), fileD (file)
	dirA := filepath.Join(tmp, "dirA")
	if err := os.Mkdir(dirA, 0o700); err != nil {
		t.Fatalf("failed creating dirA: %v", err)
	}
	fileB := filepath.Join(tmp, "fileB")
	if err := os.WriteFile(fileB, []byte("b"), 0o600); err != nil {
		t.Fatalf("failed creating fileB: %v", err)
	}
	dirC := filepath.Join(tmp, "dirC")
	if err := os.Mkdir(dirC, 0o700); err != nil {
		t.Fatalf("failed creating dirC: %v", err)
	}
	fileD := filepath.Join(tmp, "fileD")
	if err := os.WriteFile(fileD, []byte("d"), 0o600); err != nil {
		t.Fatalf("failed creating fileD: %v", err)
	}

	// Expect only directories when wantDirs = true
	dirs, err := readDirNamesFiltered(tmp, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sort.Strings(dirs)
	expectedDirs := []string{"dirA", "dirC"}
	if !reflect.DeepEqual(dirs, expectedDirs) {
		t.Fatalf("expected %v dirs, got %v", expectedDirs, dirs)
	}

	// Expect only files when wantDirs = false
	files, err := readDirNamesFiltered(tmp, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sort.Strings(files)
	expectedFiles := []string{"fileB", "fileD"}
	if !reflect.DeepEqual(files, expectedFiles) {
		t.Fatalf("expected %v files, got %v", expectedFiles, files)
	}
}
