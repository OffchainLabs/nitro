package gethexec

import (
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/rpcclient"
)

func ConfigDefaultNonSequencerTest() *Config {

	config := ConfigDefault
	config.Sequencer.Enable = false
	config.Forwarder = DefaultTestForwarderConfig
	config.ExecRPC = ExecRPCConfigTest
	config.ConsensesServer = rpcclient.TestClientConfig

	err := config.Validate()
	if err != nil {
		log.Crit("validating default config failed", "err", err)
	}
	return &config
}

func ConfigDefaultTest() *Config {
	config := ConfigDefault
	config.Sequencer = TestSequencerConfig
	config.L1Reader = headerreader.TestConfig
	config.ExecRPC = ExecRPCConfigTest
	config.ConsensesServer = rpcclient.TestClientConfig

	err := config.Validate()
	if err != nil {
		log.Crit("validating default config failed", "err", err)
	}

	return &config
}
