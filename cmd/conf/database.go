package conf

import (
	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	"os"
	"path"
	"path/filepath"
)

type PersistentConfig struct {
	GlobalConfig string `koanf:"global-config"`
	Chain        string `koanf:"chain"`
}

var PersistentConfigDefault = PersistentConfig{
	GlobalConfig: ".arbitrum",
	Chain:        "",
}

func PersistentConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.String(prefix+".global-config", PersistentConfigDefault.GlobalConfig, "directory to store global config")
	f.String(prefix+".chain", PersistentConfigDefault.Chain, "directory to store chain state")
}

func (c *PersistentConfig) ResolveDirectoryNames() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return errors.Wrap(err, "Unable to read users home directory")
	}

	// Make persistent storage directory relative to home directory if not already absolute
	if !filepath.IsAbs(c.GlobalConfig) {
		c.GlobalConfig = path.Join(homeDir, c.GlobalConfig)
	}
	err = os.MkdirAll(c.GlobalConfig, os.ModePerm)
	if err != nil {
		return errors.Wrap(err, "Unable to create global configuration directory")
	}

	// Make chain directory relative to persistent storage directory if not already absolute
	if !filepath.IsAbs(c.Chain) {
		c.Chain = path.Join(c.GlobalConfig, c.Chain)
	}
	err = os.MkdirAll(c.Chain, os.ModePerm)
	if err != nil {
		return errors.Wrap(err, "Unable to create chain directory")
	}
	if DatabaseInDirectory(c.Chain) {
		return errors.Errorf("Database in --persistent.chain (%s) directory, try specifying parent directory", c.Chain)
	}

	return nil
}

func DatabaseInDirectory(path string) bool {
	// Consider database present if file `CURRENT` in directory
	_, err := os.Stat(path + "/CURRENT")

	return err == nil
}
