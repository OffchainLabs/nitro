package rpcserver

import (
	"fmt"

	"github.com/spf13/pflag"
)

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

func ConfigAddOptions(prefix, nodeType string, f *pflag.FlagSet) {
	f.Bool(prefix+".enable", DefaultConfig.Enable, fmt.Sprintf("enable %s node to serve over rpc", nodeType))
	f.Bool(prefix+".public", DefaultConfig.Public, "rpc is public")
	f.Bool(prefix+".authenticated", DefaultConfig.Authenticated, "rpc is authenticated")
}
