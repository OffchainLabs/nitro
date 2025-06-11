package espressotee

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/pflag"
)

type TEE uint8

const (
	SGX   TEE = 0 // SGX
	NITRO TEE = 1 // AWS Nitro
)

func (t TEE) FromString(s string) (TEE, error) {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "SGX":
		return SGX, nil
	case "NITRO":
		return NITRO, nil
	default:
		return 0, fmt.Errorf("invalid TEE type: %q", s)
	}
}

type EspressoRegisterSignerConfig struct {
	MaxTxnWaitTime                time.Duration `koanf:"max-txn-wait-time"`
	RetryDelay                    time.Duration `koanf:"retry-delay"`
	MaxRetries                    uint8         `koanf:"max-retries"`
	GasLimitBufferIncreasePercent uint64        `koanf:"gas-limit-buffer-increase-percent"`
}

var DefaultEspressoRegisterSignerConfig = EspressoRegisterSignerConfig{
	MaxTxnWaitTime:                3 * time.Minute,
	RetryDelay:                    5 * time.Second,
	MaxRetries:                    5,
	GasLimitBufferIncreasePercent: 20,
}

type EspressoRegisterSignerOpts struct {
	MaxTxnWaitTime                time.Duration
	RetryDelay                    time.Duration
	MaxRetries                    int
	GasLimitBufferIncreasePercent uint64
}

func AddEspressoRegisterSignerConfigOptions(prefix string, f *pflag.FlagSet) {
	f.Duration(prefix+".max-txn-wait-time", DefaultEspressoRegisterSignerConfig.MaxTxnWaitTime, "max transaction wait time when calling espresso tee verifier contracts")
	f.Duration(prefix+".retry-delay", DefaultEspressoRegisterSignerConfig.RetryDelay, "delay in between verification calls to espresso tee contracts")
	f.Int(prefix+".max-retries", int(DefaultEspressoRegisterSignerConfig.MaxRetries), "how many times to check if we have data in our espresso tee contracts")
	f.Uint64(prefix+".gas-limit-buffer-increase-percent", DefaultEspressoRegisterSignerConfig.GasLimitBufferIncreasePercent, "buffer increase to gas limit in espresso tee contracts")
}
