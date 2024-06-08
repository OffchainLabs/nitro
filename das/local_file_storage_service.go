// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbstate/daprovider"
	"github.com/offchainlabs/nitro/das/dastree"
	"github.com/offchainlabs/nitro/util/pretty"
	flag "github.com/spf13/pflag"
	"golang.org/x/sys/unix"
)

type LocalFileStorageConfig struct {
	Enable  bool   `koanf:"enable"`
	DataDir string `koanf:"data-dir"`
}

var DefaultLocalFileStorageConfig = LocalFileStorageConfig{
	DataDir: "",
}

func LocalFileStorageConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enable", DefaultLocalFileStorageConfig.Enable, "enable storage/retrieval of sequencer batch data from a directory of files, one per batch")
	f.String(prefix+".data-dir", DefaultLocalFileStorageConfig.DataDir, "local data directory")
}

type LocalFileStorageService struct {
	dataDir string

	legacyLayout flatLayout
	layout       trieLayout

	// for testing only
	enableLegacyLayout bool
}

func NewLocalFileStorageService(dataDir string) (*LocalFileStorageService, error) {
	if unix.Access(dataDir, unix.W_OK|unix.R_OK) != nil {
		return nil, fmt.Errorf("couldn't start LocalFileStorageService, directory '%s' must be readable and writeable", dataDir)
	}
	return &LocalFileStorageService{
		dataDir:      dataDir,
		legacyLayout: flatLayout{root: dataDir},
		layout:       trieLayout{root: dataDir},
	}, nil
}

func (s *LocalFileStorageService) GetByHash(ctx context.Context, key common.Hash) ([]byte, error) {
	log.Trace("das.LocalFileStorageService.GetByHash", "key", pretty.PrettyHash(key), "this", s)
	batchPath := s.legacyLayout.batchPath(key)
	data, err := os.ReadFile(batchPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return data, nil
}

func (s *LocalFileStorageService) Put(ctx context.Context, data []byte, expiry uint64) error {
	// TODO input validation on expiry
	logPut("das.LocalFileStorageService.Store", data, expiry, s)
	key := dastree.Hash(data)
	var batchPath string
	if !s.enableLegacyLayout {
		batchPath = s.layout.batchPath(key)

		if s.layout.expiryEnabled {
			if err := createEmptyFile(s.layout.expiryPath(key, expiry)); err != nil {
				return fmt.Errorf("Couldn't create by-expiry-path index entry: %w", err)
			}
		}
	} else {
		batchPath = s.legacyLayout.batchPath(key)
	}

	// Use a temp file and rename to achieve atomic writes.
	f, err := os.CreateTemp(path.Dir(batchPath), path.Base(batchPath))
	if err != nil {
		return err
	}
	defer f.Close()
	err = f.Chmod(0o600)
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	if err != nil {
		return err
	}

	if s.enableLegacyLayout {
		tv := syscall.Timeval{
			Sec:  int64(expiry - uint64(s.legacyLayout.retention.Seconds())),
			Usec: 0,
		}
		times := []syscall.Timeval{tv, tv}
		if err = syscall.Utimes(f.Name(), times); err != nil {
			return err
		}
	}

	return os.Rename(f.Name(), batchPath)
}

func (s *LocalFileStorageService) Sync(ctx context.Context) error {
	return nil
}

func (s *LocalFileStorageService) Close(ctx context.Context) error {
	return nil
}

func (s *LocalFileStorageService) ExpirationPolicy(ctx context.Context) (daprovider.ExpirationPolicy, error) {
	return daprovider.KeepForever, nil
}

func (s *LocalFileStorageService) String() string {
	return "LocalFileStorageService(" + s.dataDir + ")"
}

func (s *LocalFileStorageService) HealthCheck(ctx context.Context) error {
	testData := []byte("Test-Data")
	err := s.Put(ctx, testData, uint64(time.Now().Add(time.Minute).Unix()))
	if err != nil {
		return err
	}
	res, err := s.GetByHash(ctx, dastree.Hash(testData))
	if err != nil {
		return err
	}
	if !bytes.Equal(res, testData) {
		return errors.New("invalid GetByHash result")
	}
	return nil
}

/*
New layout
   Access data by hash -> by-data-hash/1st octet/2nd octet/hash
   Store data with hash and expiry
   Iterate Unordered
   Iterate by Time
   Prune before


Old layout
   Access data by hash -> hash
   Iterate unordered
*/

func listDir(dir string) ([]string, error) {
	d, err := os.Open(dir)
	if err != nil {
		return nil, err
	}
	defer d.Close()

	// Read all the directory entries
	files, err := d.Readdirnames(-1)
	if err != nil {
		return nil, err
	}

	return files, nil
}

var hex64Regex = regexp.MustCompile(fmt.Sprintf("^[a-fA-F0-9]{%d}$", common.HashLength*2))

func isStorageServiceKey(key string) bool {
	return hex64Regex.MatchString(key)
}

// Copies a file by its contents to a new file, making any directories needed
// in the new file's path.
func copyFile(new, orig string) error {
	err := os.MkdirAll(path.Dir(new), 0o700)
	if err != nil {
		return fmt.Errorf("failed to create directory %s: %w", path.Dir(new), err)
	}

	origFile, err := os.Open(orig)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer origFile.Close()

	newFile, err := os.Create(new)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer newFile.Close()

	_, err = io.Copy(newFile, origFile)
	if err != nil {
		return fmt.Errorf("failed to copy contents: %w", err)
	}

	return nil
}

// Creates an empty file, making any directories needed in the new file's path.
func createEmptyFile(new string) error {
	err := os.MkdirAll(path.Dir(new), 0o700)
	if err != nil {
		return fmt.Errorf("failed to create directory %s: %w", path.Dir(new), err)
	}

	file, err := os.OpenFile(new, os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", new, err)
	}
	file.Close()
	return nil
}

func migrate(fl *flatLayout, tl *trieLayout) error {
	flIt, err := fl.iterateBatches()
	if err != nil {
		return err
	}

	if err = tl.startMigration(); err != nil {
		return err
	}

	migrationStart := time.Now()
	var migrated, skipped, removed int
	err = func() error {
		for batch, err := flIt.next(); err != io.EOF; batch, err = flIt.next() {
			if err != nil {
				return err
			}

			if tl.expiryEnabled && batch.expiry.Before(migrationStart) {
				skipped++
				continue // don't migrate expired batches
			}

			origPath := fl.batchPath(batch.key)
			newPath := tl.batchPath(batch.key)
			if err = copyFile(newPath, origPath); err != nil {
				return err
			}

			if tl.expiryEnabled {
				expiryPath := tl.expiryPath(batch.key, uint64(batch.expiry.Unix()))
				if err = createEmptyFile(expiryPath); err != nil {
					return err
				}
			}
			migrated++
		}

		return tl.commitMigration()
	}()
	if err != nil {
		return fmt.Errorf("error migrating local file store layout, retaining old layout: %w", err)
	}

	flIt, err = fl.iterateBatches()
	if err != nil {
		return err
	}
	for batch, err := flIt.next(); err != io.EOF; batch, err = flIt.next() {
		if err != nil {
			log.Warn("local file store migration completed, but error cleaning up old layout, files from that layout are now orphaned", "error", err)
			break
		}
		toRemove := fl.batchPath(batch.key)
		err = os.Remove(toRemove)
		if err != nil {
			log.Warn("local file store migration completed, but error cleaning up file from old layout, file is now orphaned", "file", toRemove, "error", err)
		}
		removed++
	}

	log.Info("Local file store legacy layout migration complete", "migratedFiles", migrated, "skippedExpiredFiles", skipped, "removedFiles", removed)

	return nil
}

func prune(tl trieLayout, pruneTil time.Time) error {
	it, err := tl.iterateBatchesByTimestamp(pruneTil)
	if err != nil {
		return err
	}
	pruned := 0
	for file, err := it.next(); err != io.EOF; file, err = it.next() {
		if err != nil {
			return err
		}
		pathByTimestamp := path.Base(file)
		key, err := DecodeStorageServiceKey(path.Base(pathByTimestamp))
		if err != nil {
			return err
		}
		pathByHash := tl.batchPath(key)
		err = recursivelyDeleteUntil(pathByHash, byDataHash)
		if err != nil {
			if os.IsNotExist(err) {
				log.Warn("Couldn't find batch to expire, it may have been previously deleted but its by-expiry-timestamp index entry still exists, trying to clean up the index next", "path", pathByHash, "indexPath", pathByTimestamp, "err", err)

			} else {
				log.Error("Couldn't prune expired batch, continuing trying to prune others", "path", pathByHash, "err", err)
				continue
			}

		}
		err = recursivelyDeleteUntil(pathByTimestamp, byExpiryTimestamp)
		if err != nil {
			log.Error("Couldn't prune expired batch expiry index entry, continuing trying to prune others", "path", pathByHash, "err", err)
		}
		pruned++
	}
	log.Info("local file store pruned expired batches", "count", pruned)
	return nil
}

func recursivelyDeleteUntil(filePath, until string) error {
	err := os.Remove(filePath)
	if err != nil {
		return err
	}

	for filePath = path.Dir(filePath); path.Base(filePath) != until; filePath = path.Dir(filePath) {
		err = os.Remove(filePath)
		if err != nil {
			if !strings.Contains(err.Error(), "directory not empty") {
				log.Warn("error cleaning up empty directory when pruning expired batches", "path", filePath, "err", err)
			}
			break
		}
	}
	return nil
}

type batchIterator interface {
	next() (batchIdentifier, bool, error)
}

type batchIdentifier struct {
	key    common.Hash
	expiry time.Time
}

const (
	defaultRetention = time.Hour * 24 * 7 * 3
)

type flatLayout struct {
	root string

	retention time.Duration
}

type flatLayoutIterator struct {
	files []string

	layout *flatLayout
}

func (l *flatLayout) batchPath(key common.Hash) string {
	return filepath.Join(l.root, EncodeStorageServiceKey(key))
}

type layerFilter func(*[][]string, int) bool

func noopFilter(*[][]string, int) bool { return true }

func (l *flatLayout) iterateBatches() (*flatLayoutIterator, error) {
	files, err := listDir(l.root)
	if err != nil {
		return nil, err
	}
	return &flatLayoutIterator{
		files:  files,
		layout: l,
	}, nil
}

func (i *flatLayoutIterator) next() (batchIdentifier, error) {
	for len(i.files) > 0 {
		var f string
		f, i.files = i.files[0], i.files[1:]
		if !isStorageServiceKey(f) {
			continue
		}
		key, err := DecodeStorageServiceKey(f)
		if err != nil {
			return batchIdentifier{}, err
		}

		fullPath := i.layout.batchPath(key)
		stat, err := os.Stat(fullPath)
		if err != nil {
			return batchIdentifier{}, err
		}

		return batchIdentifier{
			key:    key,
			expiry: stat.ModTime().Add(i.layout.retention),
		}, nil
	}
	return batchIdentifier{}, io.EOF
}

const (
	byDataHash        = "by-data-hash"
	byExpiryTimestamp = "by-expiry-timestamp"
	migratingSuffix   = "-migrating"
	expiryDivisor     = 10_000
)

type trieLayout struct {
	root          string
	expiryEnabled bool

	migrating bool // Is the trieLayout currently being migrated to
}

type trieLayoutIterator struct {
	levels  [][]string
	filters []layerFilter
	topDir  string
	layout  *trieLayout
}

func (l *trieLayout) batchPath(key common.Hash) string {
	encodedKey := EncodeStorageServiceKey(key)
	firstDir := encodedKey[:2]
	secondDir := encodedKey[2:4]

	topDir := byDataHash
	if l.migrating {
		topDir = topDir + migratingSuffix
	}

	return filepath.Join(l.root, topDir, firstDir, secondDir, encodedKey)
}

func (l *trieLayout) expiryPath(key common.Hash, expiry uint64) string {
	encodedKey := EncodeStorageServiceKey(key)
	firstDir := fmt.Sprintf("%d", expiry/expiryDivisor)
	secondDir := fmt.Sprintf("%d", expiry%expiryDivisor)

	topDir := byExpiryTimestamp
	if l.migrating {
		topDir = topDir + migratingSuffix
	}

	return filepath.Join(l.root, topDir, firstDir, secondDir, encodedKey)
}

func (l *trieLayout) iterateBatches() (*trieLayoutIterator, error) {
	var firstLevel, secondLevel, files []string
	var err error

	// TODO handle stray files that aren't dirs

	firstLevel, err = listDir(filepath.Join(l.root, byDataHash))
	if err != nil {
		return nil, err
	}

	if len(firstLevel) > 0 {
		secondLevel, err = listDir(filepath.Join(l.root, byDataHash, firstLevel[0]))
		if err != nil {
			return nil, err
		}
	}

	if len(secondLevel) > 0 {
		files, err = listDir(filepath.Join(l.root, byDataHash, firstLevel[0], secondLevel[0]))
		if err != nil {
			return nil, err
		}
	}

	storageKeyFilter := func(layers *[][]string, idx int) bool {
		return isStorageServiceKey((*layers)[idx][0])
	}

	return &trieLayoutIterator{
		levels:  [][]string{firstLevel, secondLevel, files},
		filters: []layerFilter{noopFilter, noopFilter, storageKeyFilter},
		topDir:  byDataHash,
		layout:  l,
	}, nil
}

func (l *trieLayout) iterateBatchesByTimestamp(maxTimestamp time.Time) (*trieLayoutIterator, error) {
	var firstLevel, secondLevel, files []string
	var err error

	firstLevel, err = listDir(filepath.Join(l.root, byExpiryTimestamp))
	if err != nil {
		return nil, err
	}

	if len(firstLevel) > 0 {
		secondLevel, err = listDir(filepath.Join(l.root, byExpiryTimestamp, firstLevel[0]))
		if err != nil {
			return nil, err
		}
	}

	if len(secondLevel) > 0 {
		files, err = listDir(filepath.Join(l.root, byExpiryTimestamp, firstLevel[0], secondLevel[0]))
		if err != nil {
			return nil, err
		}
	}

	beforeUpper := func(layers *[][]string, idx int) bool {
		num, err := strconv.Atoi((*layers)[idx][0])
		if err != nil {
			return false
		}
		return int64(num) <= maxTimestamp.Unix()/expiryDivisor
	}
	beforeLower := func(layers *[][]string, idx int) bool {
		num, err := strconv.Atoi((*layers)[idx-1][0] + (*layers)[idx][0])
		if err != nil {
			return false
		}
		return int64(num) <= maxTimestamp.Unix()
	}
	storageKeyFilter := func(layers *[][]string, idx int) bool {
		return isStorageServiceKey((*layers)[idx][0])
	}

	return &trieLayoutIterator{
		levels:  [][]string{firstLevel, secondLevel, files},
		filters: []layerFilter{beforeUpper, beforeLower, storageKeyFilter},
		topDir:  byExpiryTimestamp,
		layout:  l,
	}, nil
}

func (l *trieLayout) startMigration() error {
	// TODO check for existing dirs
	if !l.migrating {
		return errors.New("Local file storage already migrated to trieLayout")
	}

	if err := os.MkdirAll(filepath.Join(l.root, byDataHash+migratingSuffix), 0o700); err != nil {
		return err
	}

	if l.expiryEnabled {
		if err := os.MkdirAll(filepath.Join(l.root, byExpiryTimestamp+migratingSuffix), 0o700); err != nil {
			return err
		}
	}
	return nil

}

func (l *trieLayout) commitMigration() error {
	if !l.migrating {
		return errors.New("already finished migration")
	}

	removeSuffix := func(prefix string) error {
		oldDir := filepath.Join(l.root, prefix+migratingSuffix)
		newDir := filepath.Join(l.root, prefix)

		if err := os.Rename(oldDir, newDir); err != nil {
			return err // rename error already includes src and dst, no need to wrap
		}
		return nil
	}

	if err := removeSuffix(byDataHash); err != nil {
		return err
	}

	if l.expiryEnabled {
		if err := removeSuffix(byExpiryTimestamp); err != nil {
			return err
		}
	}

	syscall.Sync()
	return nil
}

func (it *trieLayoutIterator) next() (string, error) {
	isLeaf := func(idx int) bool {
		return idx == len(it.levels)-1
	}

	makePathAtLevel := func(idx int) string {
		pathComponents := make([]string, idx+3)
		pathComponents[0] = it.layout.root
		pathComponents[1] = it.topDir
		for i := 0; i <= idx; i++ {
			pathComponents[i+2] = it.levels[i][0]
		}
		return filepath.Join(pathComponents...)
	}

	var populateNextLevel func(idx int) error
	populateNextLevel = func(idx int) error {
		if isLeaf(idx) || len(it.levels[idx]) == 0 {
			return nil
		}
		nextLevelEntries, err := listDir(makePathAtLevel(idx))
		if err != nil {
			return err
		}
		it.levels[idx+1] = nextLevelEntries
		if len(nextLevelEntries) > 0 {
			return populateNextLevel(idx + 1)
		}
		return nil
	}

	advanceWithinLevel := func(idx int) error {
		if len(it.levels[idx]) > 1 {
			it.levels[idx] = it.levels[idx][1:]
		} else {
			it.levels[idx] = nil
		}

		return populateNextLevel(idx)
	}

	for idx := 0; idx >= 0; {
		if len(it.levels[idx]) == 0 {
			idx--
			continue
		}

		if !it.filters[idx](&it.levels, idx) {
			if err := advanceWithinLevel(idx); err != nil {
				return "", err
			}
			continue
		}

		if isLeaf(idx) {
			path := makePathAtLevel(idx)
			if err := advanceWithinLevel(idx); err != nil {
				return "", err
			}
			return path, nil
		}

		if len(it.levels[idx+1]) > 0 {
			idx++
			continue
		}

		if err := advanceWithinLevel(idx); err != nil {
			return "", err
		}
	}
	return "", io.EOF
}
