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
}

func NewLocalFileStorageService(dataDir string) (StorageService, error) {
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

func (s *LocalFileStorageService) Put(ctx context.Context, data []byte, timeout uint64) error {
	logPut("das.LocalFileStorageService.Store", data, timeout, s)
	fileName := EncodeStorageServiceKey(dastree.Hash(data))
	finalPath := s.dataDir + "/" + fileName

	// Use a temp file and rename to achieve atomic writes.
	f, err := os.CreateTemp(s.dataDir, fileName)
	if err != nil {
		return err
	}
	err = f.Chmod(0o600)
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	if err != nil {
		return err
	}
	err = f.Close()
	if err != nil {
		return err
	}

	return os.Rename(f.Name(), finalPath)

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
	files, err := d.Readdir(-1)
	if err != nil {
		return nil, err
	}

	var fileNames []string
	for _, file := range files {
		fileNames = append(fileNames, file.Name())
	}
	return fileNames, nil
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

func migrate(fl flatLayout, tl trieLayout) error {
	flIt, err := fl.iterateBatches()
	if err != nil {
		return err
	}

	if !tl.migrating {
		return errors.New("LocalFileStorage already migrated to trieLayout")
	}

	migrationStart := time.Now()

	err = func() error {
		for batch, found, err := flIt.next(); found; batch, found, err = flIt.next() {
			if err != nil {
				return err
			}

			if tl.expiryEnabled && batch.expiry.Before(migrationStart) {
				continue // don't migrate expired batches
			}

			origPath := fl.batchPath(batch.key)
			newPath := tl.batchPath(batch.key)
			if err = copyFile(newPath, origPath); err != nil {
				return err
			}

			if tl.expiryEnabled {
				expiryPath := tl.expiryPath(batch.key, uint64(batch.expiry.Unix()))
				createEmptyFile(expiryPath)
			}
		}

		return tl.commitMigration()
	}()
	if err != nil {
		return fmt.Errorf("error migrating local file store layout, retaining old layout: %w", err)
	}

	func() {
		for batch, found, err := flIt.next(); found; batch, found, err = flIt.next() {
			if err != nil {
				log.Warn("local file store migration completed, but error cleaning up old layout, files from that layout are now orphaned", "error", err)
				return
			}
			toRemove := fl.batchPath(batch.key)
			err = os.Remove(toRemove)
			if err != nil {
				log.Warn("local file store migration completed, but error cleaning up file from old layout, file is now orphaned", "file", toRemove, "error", err)
			}
		}
	}()

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

func (i *flatLayoutIterator) next() (batchIdentifier, bool, error) {
	for len(i.files) > 0 {
		var f string
		f, i.files = i.files[0], i.files[1:]
		path := filepath.Join(i.layout.root, f)
		if !isStorageServiceKey(f) {
			log.Warn("Incorrectly named batch file found, ignoring", "file", path)
			continue
		}
		key, err := DecodeStorageServiceKey(f)

		stat, err := os.Stat(f)
		if err != nil {
			return batchIdentifier{}, false, err
		}

		return batchIdentifier{
			key:    key,
			expiry: stat.ModTime().Add(i.layout.retention),
		}, true, nil
	}
	return batchIdentifier{}, false, nil
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
	firstLevel  []string
	secondLevel []string
	files       []string

	layout *trieLayout
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

func (l *trieLayout) expiryPath(key common.Hash, expiry uint64) pathParts {
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

	return &trieLayoutIterator{
		firstLevel:  firstLevel,
		secondLevel: secondLevel,
		files:       files,
		layout:      l,
	}, nil
}

func (i *trieLayout) commitMigration() error {
	if !i.migrating {
		return errors.New("already finished migration")
	}

	oldDir := filepath.Join(i.root, byDataHash+migratingSuffix)
	newDir := filepath.Join(i.root, byDataHash)

	if err := os.Rename(oldDir, newDir); err != nil {
		return fmt.Errorf("couldn't rename \"%s\" to \"%s\": %w", oldDir, newDir, err)
	}

	syscall.Sync()
	return nil
}

func (i *trieLayoutIterator) next() (string, bool, error) {
	for len(i.firstLevel) > 0 {
		for len(i.secondLevel) > 0 {
			if len(i.files) > 0 {
				var f string
				f, i.files = i.files[0], i.files[1:]
				return filepath.Join(i.layout.root, byDataHash, i.firstLevel[0], i.secondLevel[0], f), true, nil
			}

			if len(i.secondLevel) <= 1 {
				return "", false, nil
			}
			i.secondLevel = i.secondLevel[1:]

			files, err := listDir(filepath.Join(i.layout.root, byDataHash, i.firstLevel[0], i.secondLevel[0]))
			if err != nil {
				return "", false, err
			}
			i.files = files
		}

		if len(i.firstLevel) <= 1 {
			return "", false, nil
		}
		i.firstLevel = i.firstLevel[1:]
		secondLevel, err := listDir(filepath.Join(i.layout.root, byDataHash, i.firstLevel[0]))
		if err != nil {
			return "", false, err
		}
		i.secondLevel = secondLevel
	}

	return "", false, nil
}
