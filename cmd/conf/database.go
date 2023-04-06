// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

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
	Handles      int    `koanf:"handles"`
	Ancient      string `koanf:"ancient"`
}

var PersistentConfigDefault = PersistentConfig{
	GlobalConfig: ".arbitrum",
	Chain:        "",
	Handles:      512,
	Ancient:      "",
}

func PersistentConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.String(prefix+".global-config", PersistentConfigDefault.GlobalConfig, "directory to store global config")
	f.String(prefix+".chain", PersistentConfigDefault.Chain, "directory to store chain state")
	f.Int(prefix+".handles", PersistentConfigDefault.Handles, "number of file descriptor handles to use for the database")
	f.String(prefix+".ancient", PersistentConfigDefault.Ancient, "directory of ancient where the chain freezer can be opened")
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

	return nil
}

func DatabaseInDirectory(path string) bool {
	// Consider database present if file `CURRENT` in directory
	_, err := os.Stat(path + "/CURRENT")

	return err == nil
}
