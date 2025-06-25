package espressotee

import (
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/log"
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

type ContractVerificationFunc func() (bool, error)

func ContractVerification(
	maxRetries int,
	retryDelay time.Duration,
	fn ContractVerificationFunc,
	msg string,
) (bool, error) {
	var err error
	success := false
	for attempt := 0; attempt < maxRetries; attempt++ {
		success, err = fn()
		if err != nil {
			log.Error(msg, "err", err)
		}
		if success {
			return true, nil
		}

		if attempt < maxRetries-1 {
			log.Error(msg, "attempt", attempt, "retry delay", retryDelay)
			time.Sleep(retryDelay)
		}
	}
	return false, nil
}

type BaseFeeCheckFunc func() (*big.Int, error)

func BaseFeeCheck(
	maxBaseFee uint64,
	maxRetries int,
	retryDelay time.Duration,
	fn BaseFeeCheckFunc,
	msg string,
) error {
	lowBaseFee := false
	for attempt := 0; attempt < maxRetries; attempt++ {
		latestBaseFee, err := fn()
		if err != nil && attempt < maxRetries-1 {
			log.Error(msg, "err", err, "delay", retryDelay, "attempt", attempt+1)
			if attempt < maxRetries-1 {
				time.Sleep(retryDelay)
			}
			continue
		}

		if latestBaseFee.Uint64() > maxBaseFee {
			log.Error(
				msg,
				"base fee", latestBaseFee.Uint64(),
				"max base fee", maxBaseFee,
				"delay", retryDelay,
				"attempt", attempt+1,
			)
			if attempt < maxRetries-1 {
				time.Sleep(retryDelay)
			}
			continue
		}

		lowBaseFee = true
		break
	}
	if !lowBaseFee {
		return fmt.Errorf("base fee is not low enough to attempt to register signer")
	}
	return nil
}

type EspressoRegisterSignerConfig struct {
	MaxTxnWaitTime                time.Duration `koanf:"max-txn-wait-time"`
	RetryBaseFeeDelay             time.Duration `koanf:"retry-base-fee-delay"`
	RetryReadContractDelay        time.Duration `koanf:"retry-read-contract-delay"`
	MaxRetries                    uint8         `koanf:"max-retries"`
	GasLimitBufferIncreasePercent uint64        `koanf:"gas-limit-buffer-increase-percent"`
	MaxBaseFee                    uint64        `koanf:"max-base-fee"`
}

var DefaultEspressoRegisterSignerConfig = EspressoRegisterSignerConfig{
	MaxTxnWaitTime:                3 * time.Minute,
	RetryBaseFeeDelay:             1 * time.Minute,
	RetryReadContractDelay:        5 * time.Second,
	MaxRetries:                    5,
	GasLimitBufferIncreasePercent: 20,
	MaxBaseFee:                    70000000,
}

type EspressoRegisterSignerOpts struct {
	MaxTxnWaitTime                time.Duration
	RetryBaseFeeDelay             time.Duration
	RetryReadContractDelay        time.Duration
	MaxRetries                    int
	GasLimitBufferIncreasePercent uint64
	MaxBaseFee                    uint64
}

func AddEspressoRegisterSignerConfigOptions(prefix string, f *pflag.FlagSet) {
	f.Duration(prefix+".max-txn-wait-time", DefaultEspressoRegisterSignerConfig.MaxTxnWaitTime, "max transaction wait time when calling espresso tee verifier contracts")
	f.Duration(prefix+".retry-base-fee-delay", DefaultEspressoRegisterSignerConfig.RetryBaseFeeDelay, "delay in calls to check the base fee")
	f.Duration(prefix+".retry-read-contract-delay", DefaultEspressoRegisterSignerConfig.RetryReadContractDelay, "delay in calls to read from contract for verification")
	f.Int(prefix+".max-retries", int(DefaultEspressoRegisterSignerConfig.MaxRetries), "how many times to check if we have data in our espresso tee contracts")
	f.Uint64(prefix+".gas-limit-buffer-increase-percent", DefaultEspressoRegisterSignerConfig.GasLimitBufferIncreasePercent, "buffer increase to gas limit in espresso tee contracts")
	f.Uint64(prefix+".max-base-fee", DefaultEspressoRegisterSignerConfig.MaxBaseFee, "max base fee to use when calling espresso tee contracts")
}
