// DANGER! this file is included in all builds
// DANGER! do not place any of the experimental logic and features here

package debugblock

type Config struct {
	OverwriteChainConfig bool   `koanf:"overwrite-chain-config"`
	DebugAddress         string `koanf:"debug-address"`
	DebugBlockNum        uint64 `koanf:"debug-blocknum"`
}

var ConfigDefault = Config{
	OverwriteChainConfig: false,
	DebugAddress:         "",
	DebugBlockNum:        0,
}
