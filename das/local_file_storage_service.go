// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"context"
	"encoding/base32"
	"fmt"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbstate"
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
}

func NewLocalFileStorageService(dataDir string) (StorageService, error) {
	if unix.Access(dataDir, unix.W_OK|unix.R_OK) != nil {
		return nil, fmt.Errorf("Couldn't start LocalFileStorageService, directory '%s' must be readable and writeable", dataDir)
	}
	return &LocalFileStorageService{dataDir: dataDir}, nil
}

func (s *LocalFileStorageService) GetByHash(ctx context.Context, key []byte) ([]byte, error) {
	log.Trace("das.LocalFileStorageService.GetByHash", "key", pretty.FirstFewBytes(key), "this", s)
	pathname := s.dataDir + "/" + base32.StdEncoding.EncodeToString(key)
	return os.ReadFile(pathname)
}

func (s *LocalFileStorageService) Put(ctx context.Context, data []byte, timeout uint64) error {
	log.Trace("das.LocalFileStorageService.Store", "message", pretty.FirstFewBytes(data), "timeout", time.Unix(int64(timeout), 0), "this", s)
	fileName := base32.StdEncoding.EncodeToString(crypto.Keccak256(data))
	finalPath := s.dataDir + "/" + fileName

	// Use a temp file and rename to achieve atomic writes.
	f, err := os.CreateTemp(s.dataDir, fileName)
	if err != nil {
		return err
	}
	err = f.Chmod(0600)
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

func (s *LocalFileStorageService) ExpirationPolicy(ctx context.Context) arbstate.ExpirationPolicy {
	return arbstate.KeepForever
}

func (s *LocalFileStorageService) String() string {
	return "LocalFileStorageService(" + s.dataDir + ")"
}

func (s *LocalFileStorageService) HealthCheck(ctx context.Context) error {
	return nil
}
