package rpcserver

import "github.com/spf13/pflag"

type Config struct {
	Enable        bool `koanf:"enable"`
	Public        bool `koanf:"public"`
	Authenticated bool `koanf:"authenticated"`
}

var DefaultConfig = Config{
	Enable:        false,
	Public:        false,
	Authenticated: true,
}

func ConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".enable", DefaultConfig.Enable, "enable execution node to serve over rpc")
	f.Bool(prefix+".public", DefaultConfig.Public, "rpc is public")
	f.Bool(prefix+".authenticated", DefaultConfig.Authenticated, "rpc is authenticated")
}
