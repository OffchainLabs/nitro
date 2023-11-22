// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package genericconf

import (
	"path"
	"path/filepath"

	flag "github.com/spf13/pflag"
)

const PASSWORD_NOT_SET = "PASSWORD_NOT_SET"

type WalletConfig struct {
	Pathname      string `koanf:"pathname"`
	Password      string `koanf:"password"`
	PrivateKey    string `koanf:"private-key"`
	Account       string `koanf:"account"`
	OnlyCreateKey bool   `koanf:"only-create-key"`
}

func (w *WalletConfig) Pwd() *string {
	if w.Password == PASSWORD_NOT_SET {
		return nil
	}
	return &w.Password
}

var WalletConfigDefault = WalletConfig{
	Pathname:      "",
	Password:      PASSWORD_NOT_SET,
	PrivateKey:    "",
	Account:       "",
	OnlyCreateKey: false,
}

func WalletConfigAddOptions(prefix string, f *flag.FlagSet, defaultPathname string) {
	f.String(prefix+".pathname", defaultPathname, "pathname for wallet")
	f.String(prefix+".password", WalletConfigDefault.Password, "wallet passphrase")
	f.String(prefix+".private-key", WalletConfigDefault.PrivateKey, "private key for wallet")
	f.String(prefix+".account", WalletConfigDefault.Account, "account to use (default is first account in keystore)")
	f.Bool(prefix+".only-create-key", WalletConfigDefault.OnlyCreateKey, "if true, creates new key then exits")
}

func (w *WalletConfig) ResolveDirectoryNames(chain string) {
	// Make wallet directories relative to chain directory if specified and not already absolute
	if len(w.Pathname) != 0 && !filepath.IsAbs(w.Pathname) {
		w.Pathname = path.Join(chain, w.Pathname)
	}
}
