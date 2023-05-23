// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/offchainlabs/nitro/util/colors"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func TestSeqConfig(t *testing.T) {
	args := strings.Split("--persistent.chain /tmp/data --init.dev-init --node.parent-chain-reader.enable=false --parent-chain.id 5 --chain.id 421613 --parent-chain.wallet.pathname /l1keystore --parent-chain.wallet.password passphrase --http.addr 0.0.0.0 --ws.addr 0.0.0.0 --node.sequencer.enable --node.feed.output.enable --node.feed.output.port 9642", " ")
	_, _, _, err := ParseNode(context.Background(), args)
	Require(t, err)
}

func TestUnsafeStakerConfig(t *testing.T) {
	args := strings.Split("--persistent.chain /tmp/data --init.dev-init --node.parent-chain-reader.enable=false --parent-chain.id 5 --chain.id 421613 --parent-chain.wallet.pathname /l1keystore --parent-chain.wallet.password passphrase --http.addr 0.0.0.0 --ws.addr 0.0.0.0 --node.staker.enable --node.staker.strategy MakeNodes --node.staker.staker-interval 10s --node.forwarding-target null --node.staker.dangerous.without-block-validator", " ")
	_, _, _, err := ParseNode(context.Background(), args)
	Require(t, err)
}

func TestValidatorConfig(t *testing.T) {
	args := strings.Split("--persistent.chain /tmp/data --init.dev-init --node.parent-chain-reader.enable=false --parent-chain.id 5 --chain.id 421613 --parent-chain.wallet.pathname /l1keystore --parent-chain.wallet.password passphrase --http.addr 0.0.0.0 --ws.addr 0.0.0.0 --node.staker.enable --node.staker.strategy MakeNodes --node.staker.staker-interval 10s --node.forwarding-target null", " ")
	_, _, _, err := ParseNode(context.Background(), args)
	Require(t, err)
}

func TestAggregatorConfig(t *testing.T) {
	args := strings.Split("--persistent.chain /tmp/data --init.dev-init --node.parent-chain-reader.enable=false --parent-chain.id 5 --chain.id 421613 --parent-chain.wallet.pathname /l1keystore --parent-chain.wallet.password passphrase --http.addr 0.0.0.0 --ws.addr 0.0.0.0 --node.sequencer.enable --node.feed.output.enable --node.feed.output.port 9642 --node.data-availability.enable --node.data-availability.rpc-aggregator.backends {[\"url\":\"http://localhost:8547\",\"pubkey\":\"abc==\",\"signerMask\":0x1]}", " ")
	_, _, _, err := ParseNode(context.Background(), args)
	Require(t, err)
}

func TestReloads(t *testing.T) {
	var check func(node reflect.Value, cold bool, path string)
	check = func(node reflect.Value, cold bool, path string) {
		if node.Kind() != reflect.Struct {
			return
		}

		for i := 0; i < node.NumField(); i++ {
			hot := node.Type().Field(i).Tag.Get("reload") == "hot"
			dot := path + "." + node.Type().Field(i).Name
			if hot && cold {
				t.Fatalf(fmt.Sprintf(
					"Option %v%v%v is reloadable but %v%v%v is not",
					colors.Red, dot, colors.Clear,
					colors.Red, path, colors.Clear,
				))
			}
			if hot {
				colors.PrintBlue(dot)
			}
			check(node.Field(i), !hot, dot)
		}
	}

	config := NodeConfigDefault
	update := NodeConfigDefault
	update.Node.Sequencer.MaxBlockSpeed++

	check(reflect.ValueOf(config), false, "config")
	Require(t, config.CanReload(&config))
	Require(t, config.CanReload(&update))

	testUnsafe := func() {
		if config.CanReload(&update) == nil {
			Fail(t, "failed to detect unsafe reload")
		}
		update = NodeConfigDefault
	}

	// check that non-reloadable fields fail assignment
	update.Metrics = !update.Metrics
	testUnsafe()
	update.L2.ChainID++
	testUnsafe()
	update.Node.Sequencer.Forwarder.ConnectionTimeout++
	testUnsafe()
}

func TestLiveNodeConfig(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// create a config file
	configFile := filepath.Join(t.TempDir(), "config.json")
	jsonConfig := "{\"chain\":{\"id\":421613}}"
	Require(t, WriteToConfigFile(configFile, jsonConfig))

	args := strings.Split("--file-logging.enable=false --persistent.chain /tmp/data --init.dev-init --node.parent-chain-reader.enable=false --parent-chain.id 5 --parent-chain.wallet.pathname /l1keystore --parent-chain.wallet.password passphrase --http.addr 0.0.0.0 --ws.addr 0.0.0.0 --node.sequencer.enable --node.feed.output.enable --node.feed.output.port 9642", " ")
	args = append(args, []string{"--conf.file", configFile}...)
	config, _, _, err := ParseNode(context.Background(), args)
	Require(t, err)

	liveConfig := NewLiveNodeConfig(args, config, func(path string) string { return path })

	// check updating the config
	update := config.ShallowClone()
	expected := config.ShallowClone()
	update.Node.Sequencer.MaxBlockSpeed += 2 * time.Millisecond
	expected.Node.Sequencer.MaxBlockSpeed += 2 * time.Millisecond
	Require(t, liveConfig.set(update))
	if !reflect.DeepEqual(liveConfig.get(), expected) {
		Fail(t, "failed to set config")
	}

	// check that an invalid reload gets rejected
	update = config.ShallowClone()
	update.L2.ChainID++
	if liveConfig.set(update) == nil {
		Fail(t, "failed to reject invalid update")
	}
	if !reflect.DeepEqual(liveConfig.get(), expected) {
		Fail(t, "config should not change if its update fails")
	}

	// starting the LiveNodeConfig after testing LiveNodeConfig.set to avoid race condition in the test
	liveConfig.Start(ctx)

	// reload config
	expected = config.ShallowClone()
	Require(t, syscall.Kill(syscall.Getpid(), syscall.SIGUSR1))
	if !PollLiveConfigUntilEqual(liveConfig, expected) {
		Fail(t, "live config differs from expected")
	}

	// check that reloading the config again doesn't change anything
	Require(t, syscall.Kill(syscall.Getpid(), syscall.SIGUSR1))
	time.Sleep(80 * time.Millisecond)
	if !reflect.DeepEqual(liveConfig.get(), expected) {
		Fail(t, "live config differs from expected")
	}

	// change the config file
	expected = config.ShallowClone()
	expected.Node.Sequencer.MaxBlockSpeed += time.Millisecond
	jsonConfig = fmt.Sprintf("{\"node\":{\"sequencer\":{\"max-block-speed\":\"%s\"}}, \"chain\":{\"id\":421613}}", expected.Node.Sequencer.MaxBlockSpeed.String())
	Require(t, WriteToConfigFile(configFile, jsonConfig))

	// trigger LiveConfig reload
	Require(t, syscall.Kill(syscall.Getpid(), syscall.SIGUSR1))

	if !PollLiveConfigUntilEqual(liveConfig, expected) {
		Fail(t, "failed to update config", config.Node.Sequencer.MaxBlockSpeed, update.Node.Sequencer.MaxBlockSpeed)
	}

	// change chain.id in the config file (currently non-reloadable)
	jsonConfig = fmt.Sprintf("{\"node\":{\"sequencer\":{\"max-block-speed\":\"%s\"}}, \"chain\":{\"id\":421703}}", expected.Node.Sequencer.MaxBlockSpeed.String())
	Require(t, WriteToConfigFile(configFile, jsonConfig))

	// trigger LiveConfig reload
	Require(t, syscall.Kill(syscall.Getpid(), syscall.SIGUSR1))

	if PollLiveConfigUntilNotEqual(liveConfig, expected) {
		Fail(t, "failed to reject invalid update")
	}
}

func TestPeriodicReloadOfLiveNodeConfig(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// create config file with ReloadInterval = 20 ms
	configFile := filepath.Join(t.TempDir(), "config.json")
	jsonConfig := "{\"conf\":{\"reload-interval\":\"20ms\"}}"
	Require(t, WriteToConfigFile(configFile, jsonConfig))

	args := strings.Split("--persistent.chain /tmp/data --init.dev-init --node.parent-chain-reader.enable=false --parent-chain.id 5 --chain.id 421613 --parent-chain.wallet.pathname /l1keystore --parent-chain.wallet.password passphrase --http.addr 0.0.0.0 --ws.addr 0.0.0.0 --node.sequencer.enable --node.feed.output.enable --node.feed.output.port 9642", " ")
	args = append(args, []string{"--conf.file", configFile}...)
	config, _, _, err := ParseNode(context.Background(), args)
	Require(t, err)

	liveConfig := NewLiveNodeConfig(args, config, func(path string) string { return path })
	liveConfig.Start(ctx)

	// test if periodic reload works
	expected := config.ShallowClone()
	expected.Conf.ReloadInterval = 0
	jsonConfig = "{\"conf\":{\"reload-interval\":\"0\"}}"
	Require(t, WriteToConfigFile(configFile, jsonConfig))
	start := time.Now()
	if !PollLiveConfigUntilEqual(liveConfig, expected) {
		Fail(t, fmt.Sprintf("failed to update config after %d ms, while reload interval is %s", time.Since(start).Milliseconds(), config.Conf.ReloadInterval))
	}

	// test if previous config successfully disabled periodic reload
	expected = config.ShallowClone()
	expected.Conf.ReloadInterval = 10 * time.Millisecond
	jsonConfig = "{\"conf\":{\"reload-interval\":\"10ms\"}}"
	Require(t, WriteToConfigFile(configFile, jsonConfig))
	time.Sleep(80 * time.Millisecond)
	if reflect.DeepEqual(liveConfig.get(), expected) {
		Fail(t, "failed to disable periodic reload")
	}
}

func WriteToConfigFile(path string, jsonConfig string) error {
	return os.WriteFile(path, []byte(jsonConfig), 0600)
}

func PollLiveConfigUntilEqual(liveConfig *LiveNodeConfig, expected *NodeConfig) bool {
	return PollLiveConfig(liveConfig, expected, true)
}
func PollLiveConfigUntilNotEqual(liveConfig *LiveNodeConfig, expected *NodeConfig) bool {
	return PollLiveConfig(liveConfig, expected, false)
}

func PollLiveConfig(liveConfig *LiveNodeConfig, expected *NodeConfig, equal bool) bool {
	for i := 0; i < 16; i++ {
		if reflect.DeepEqual(liveConfig.get(), expected) == equal {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

func Require(t *testing.T, err error, text ...interface{}) {
	t.Helper()
	testhelpers.RequireImpl(t, err, text...)
}

func Fail(t *testing.T, printables ...interface{}) {
	t.Helper()
	testhelpers.FailImpl(t, printables...)
}
