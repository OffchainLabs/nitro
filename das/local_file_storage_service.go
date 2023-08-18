// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"bytes"
	"context"
	"encoding/base32"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/das/dastree"
	"github.com/offchainlabs/nitro/util/pretty"
	flag "github.com/spf13/pflag"
	"golang.org/x/sys/unix"
)

type LocalFileStorageConfig struct {
	Enable                 bool   `koanf:"enable"`
	DataDir                string `koanf:"data-dir"`
	SyncFromStorageService bool   `koanf:"sync-from-storage-service"`
	SyncToStorageService   bool   `koanf:"sync-to-storage-service"`
}

var DefaultLocalFileStorageConfig = LocalFileStorageConfig{
	DataDir: "",
}

func LocalFileStorageConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enable", DefaultLocalFileStorageConfig.Enable, "enable storage/retrieval of sequencer batch data from a directory of files, one per batch")
	f.String(prefix+".data-dir", DefaultLocalFileStorageConfig.DataDir, "local data directory")
	f.Bool(prefix+".sync-from-storage-service", DefaultLocalFileStorageConfig.SyncFromStorageService, "enable local storage to be used as a source for regular sync storage")
	f.Bool(prefix+".sync-to-storage-service", DefaultLocalFileStorageConfig.SyncToStorageService, "enable local storage to be used as a sink for regular sync storage")
}

type LocalFileStorageService struct {
	dataDir string
}

func NewLocalFileStorageService(dataDir string) (StorageService, error) {
	if unix.Access(dataDir, unix.W_OK|unix.R_OK) != nil {
		return nil, fmt.Errorf("couldn't start LocalFileStorageService, directory '%s' must be readable and writeable", dataDir)
	}
	return &LocalFileStorageService{dataDir: dataDir}, nil
}

func (s *LocalFileStorageService) GetByHash(ctx context.Context, key common.Hash) ([]byte, error) {
	log.Trace("das.LocalFileStorageService.GetByHash", "key", pretty.PrettyHash(key), "this", s)
	pathname := s.dataDir + "/" + EncodeStorageServiceKey(key)
	data, err := os.ReadFile(pathname)
	if err != nil {
		// Just for backward compatability.
		pathname = s.dataDir + "/" + base32.StdEncoding.EncodeToString(key.Bytes())
		data, err = os.ReadFile(pathname)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil, ErrNotFound
			}
			return nil, err
		}
		return data, nil
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

func (s *LocalFileStorageService) putKeyValue(ctx context.Context, key common.Hash, value []byte) error {
	fileName := EncodeStorageServiceKey(key)
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
	_, err = f.Write(value)
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

func (s *LocalFileStorageService) ExpirationPolicy(ctx context.Context) (arbstate.ExpirationPolicy, error) {
	return arbstate.KeepForever, nil
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
