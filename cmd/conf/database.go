// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package conf

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	flag "github.com/spf13/pflag"
)

type PersistentConfig struct {
	GlobalConfig string `koanf:"global-config"`
	Chain        string `koanf:"chain"`
	LogDir       string `koanf:"log-dir"`
	Handles      int    `koanf:"handles"`
	Ancient      string `koanf:"ancient"`
	DBEngine     string `koanf:"db-engine"`
}

var PersistentConfigDefault = PersistentConfig{
	GlobalConfig: ".arbitrum",
	Chain:        "",
	LogDir:       "",
	Handles:      512,
	Ancient:      "",
	DBEngine:     "leveldb",
}

func PersistentConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.String(prefix+".global-config", PersistentConfigDefault.GlobalConfig, "directory to store global config")
	f.String(prefix+".chain", PersistentConfigDefault.Chain, "directory to store chain state")
	f.String(prefix+".log-dir", PersistentConfigDefault.LogDir, "directory to store log file")
	f.Int(prefix+".handles", PersistentConfigDefault.Handles, "number of file descriptor handles to use for the database")
	f.String(prefix+".ancient", PersistentConfigDefault.Ancient, "directory of ancient where the chain freezer can be opened")
	f.String(prefix+".db-engine", PersistentConfigDefault.DBEngine, "backing database implementation to use ('leveldb' or 'pebble')")
}

func (c *PersistentConfig) ResolveDirectoryNames() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("unable to read users home directory: %w", err)
	}

	// Make persistent storage directory relative to home directory if not already absolute
	if !filepath.IsAbs(c.GlobalConfig) {
		c.GlobalConfig = path.Join(homeDir, c.GlobalConfig)
	}
	err = os.MkdirAll(c.GlobalConfig, os.ModePerm)
	if err != nil {
		return fmt.Errorf("unable to create global configuration directory: %w", err)
	}

	// Make chain directory relative to persistent storage directory if not already absolute
	if !filepath.IsAbs(c.Chain) {
		c.Chain = path.Join(c.GlobalConfig, c.Chain)
	}
	err = os.MkdirAll(c.Chain, os.ModePerm)
	if err != nil {
		return fmt.Errorf("unable to create chain directory: %w", err)
	}
	if DatabaseInDirectory(c.Chain) {
		return fmt.Errorf("database in --persistent.chain (%s) directory, try specifying parent directory", c.Chain)
	}

	// Make Log directory relative to persistent storage directory if not already absolute
	if !filepath.IsAbs(c.LogDir) {
		c.LogDir = path.Join(c.Chain, c.LogDir)
	}
	if c.LogDir != c.Chain {
		err = os.MkdirAll(c.LogDir, os.ModePerm)
		if err != nil {
			return fmt.Errorf("unable to create Log directory: %w", err)
		}
		if DatabaseInDirectory(c.LogDir) {
			return fmt.Errorf("database in --persistent.log-dir (%s) directory, try specifying parent directory", c.LogDir)
		}
	}
	return nil
}

func DatabaseInDirectory(path string) bool {
	// Consider database present if file `CURRENT` in directory
	_, err := os.Stat(path + "/CURRENT")

	return err == nil
}

func (c *PersistentConfig) Validate() error {
	// we are validating .db-engine here to avoid unintended behaviour as empty string value also has meaning in geth's node.Config.DBEngine
	if c.DBEngine != "leveldb" && c.DBEngine != "pebble" {
		return fmt.Errorf(`invalid .db-engine choice: %q, allowed "leveldb" or "pebble"`, c.DBEngine)
	}
	return nil
}
