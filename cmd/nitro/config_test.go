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

	"github.com/r3labs/diff/v3"
	flag "github.com/spf13/pflag"

	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/cmd/util/confighelpers"
	"github.com/offchainlabs/nitro/das"
	"github.com/offchainlabs/nitro/util/colors"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func TestEmptyCliConfig(t *testing.T) {
	f := flag.NewFlagSet("", flag.ContinueOnError)
	NodeConfigAddOptions(f)
	k, err := confighelpers.BeginCommonParse(f, []string{})
	Require(t, err)
	err = das.FixKeysetCLIParsing("node.data-availability.rpc-aggregator.backends", k)
	Require(t, err)
	var emptyCliNodeConfig NodeConfig
	err = confighelpers.EndCommonParse(k, &emptyCliNodeConfig)
	Require(t, err)
	if !reflect.DeepEqual(emptyCliNodeConfig, NodeConfigDefault) {
		changelog, err := diff.Diff(emptyCliNodeConfig, NodeConfigDefault)
		Require(t, err)
		Fail(t, "empty cli config differs from expected default", changelog)
	}
}

func TestSeqConfig(t *testing.T) {
	args := strings.Split("--persistent.chain /tmp/data --init.dev-init --node.parent-chain-reader.enable=false --parent-chain.id 5 --chain.id 421613 --node.batch-poster.parent-chain-wallet.pathname /l1keystore --node.batch-poster.parent-chain-wallet.password passphrase --http.addr 0.0.0.0 --ws.addr 0.0.0.0 --node.sequencer --execution.sequencer.enable --node.feed.output.enable --node.feed.output.port 9642 --node.transaction-streamer.track-block-metadata-from=10", " ")
	_, _, err := ParseNode(context.Background(), args)
	Require(t, err)
}

func TestUnsafeStakerConfig(t *testing.T) {
	args := strings.Split("--persistent.chain /tmp/data --init.dev-init --node.parent-chain-reader.enable=false --parent-chain.id 5 --chain.id 421613 --node.staker.parent-chain-wallet.pathname /l1keystore --node.staker.parent-chain-wallet.password passphrase --http.addr 0.0.0.0 --ws.addr 0.0.0.0 --node.staker.enable --node.staker.strategy MakeNodes --node.staker.staker-interval 10s --execution.forwarding-target null --node.staker.dangerous.without-block-validator", " ")
	_, _, err := ParseNode(context.Background(), args)
	Require(t, err)
}

const validatorArgs = "--persistent.chain /tmp/data --init.dev-init --node.parent-chain-reader.enable=false --parent-chain.id 5 --chain.id 421613 --node.staker.parent-chain-wallet.pathname /l1keystore --node.staker.parent-chain-wallet.password passphrase --http.addr 0.0.0.0 --ws.addr 0.0.0.0 --node.staker.enable --node.staker.strategy MakeNodes --node.staker.staker-interval 10s --execution.forwarding-target null"

func TestValidatorConfig(t *testing.T) {
	args := strings.Split("--persistent.chain /tmp/data --init.dev-init --node.parent-chain-reader.enable=false --parent-chain.id 5 --chain.id 421613 --node.staker.parent-chain-wallet.pathname /l1keystore --node.staker.parent-chain-wallet.password passphrase --http.addr 0.0.0.0 --ws.addr 0.0.0.0 --node.staker.enable --node.staker.strategy MakeNodes --node.staker.staker-interval 10s --execution.forwarding-target null", " ")
	_, _, err := ParseNode(context.Background(), args)
	Require(t, err)
}

func TestInvalidCachingStateSchemeForValidator(t *testing.T) {
	validatorArgsWithPathScheme := fmt.Sprintf("%s --execution.caching.state-scheme path", validatorArgs)
	args := strings.Split(validatorArgsWithPathScheme, " ")
	_, _, err := ParseNode(context.Background(), args)
	if !strings.Contains(err.Error(), "path cannot be used as execution.caching.state-scheme when validator is required") {
		Fail(t, "failed to detect invalid state scheme for validator")
	}
}

func TestInvalidArchiveConfig(t *testing.T) {
	args := strings.Split("--execution.caching.archive --execution.caching.state-scheme path --persistent.chain /tmp/data --init.dev-init --node.parent-chain-reader.enable=false --parent-chain.id 5 --chain.id 421613 --node.staker.parent-chain-wallet.pathname /l1keystore --node.staker.parent-chain-wallet.password passphrase --http.addr 0.0.0.0 --ws.addr 0.0.0.0 --node.staker.enable --node.staker.strategy MakeNodes --node.staker.staker-interval 10s --execution.forwarding-target null", " ")
	_, _, err := ParseNode(context.Background(), args)
	if !strings.Contains(err.Error(), "archive cannot be set when using path as the state-scheme") {
		Fail(t, "failed to detect invalid state scheme for archive")
	}
}

func TestAggregatorConfig(t *testing.T) {
	args := strings.Split("--persistent.chain /tmp/data --init.dev-init --node.parent-chain-reader.enable=false --parent-chain.id 5 --chain.id 421613 --node.batch-poster.parent-chain-wallet.pathname /l1keystore --node.batch-poster.parent-chain-wallet.password passphrase --http.addr 0.0.0.0 --ws.addr 0.0.0.0 --node.sequencer --execution.sequencer.enable --node.feed.output.enable --node.feed.output.port 9642 --node.data-availability.enable --node.data-availability.rpc-aggregator.backends [{\"url\":\"http://localhost:8547\",\"pubkey\":\"abc==\"}] --node.transaction-streamer.track-block-metadata-from=10", " ")
	_, _, err := ParseNode(context.Background(), args)
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
				t.Fatalf(
					"Option %v%v%v is reloadable but %v%v%v is not",
					colors.Red, dot, colors.Clear,
					colors.Red, path, colors.Clear,
				)
			}
			if hot {
				colors.PrintBlue(dot)
			}
			check(node.Field(i), !hot, dot)
		}
	}

	config := NodeConfigDefault
	update := NodeConfigDefault
	update.Node.BatchPoster.MaxSize++

	check(reflect.ValueOf(config), false, "config")
	Require(t, config.CanReload(&config))
	Require(t, config.CanReload(&update))

	testUnsafe := func() {
		t.Helper()
		if config.CanReload(&update) == nil {
			Fail(t, "failed to detect unsafe reload")
		}
		update = NodeConfigDefault
	}

	// check that non-reloadable fields fail assignment
	update.Metrics = !update.Metrics
	testUnsafe()
	update.ParentChain.ID++
	testUnsafe()
	update.Node.Staker.Enable = !update.Node.Staker.Enable
	testUnsafe()
}

func TestLiveNodeConfig(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// create a config file
	configFile := filepath.Join(t.TempDir(), "config.json")
	jsonConfig := "{\"chain\":{\"id\":421613}}"
	Require(t, WriteToConfigFile(configFile, jsonConfig))

	args := strings.Split("--file-logging.enable=false --persistent.chain /tmp/data --init.dev-init --node.parent-chain-reader.enable=false --parent-chain.id 5 --node.batch-poster.parent-chain-wallet.pathname /l1keystore --node.batch-poster.parent-chain-wallet.password passphrase --http.addr 0.0.0.0 --ws.addr 0.0.0.0 --node.sequencer --execution.sequencer.enable --node.feed.output.enable --node.feed.output.port 9642 --node.transaction-streamer.track-block-metadata-from=10", " ")
	args = append(args, []string{"--conf.file", configFile}...)
	config, _, err := ParseNode(context.Background(), args)
	Require(t, err)

	liveConfig := genericconf.NewLiveConfig[*NodeConfig](args, config, func(ctx context.Context, args []string) (*NodeConfig, error) {
		nodeConfig, _, err := ParseNode(ctx, args)
		return nodeConfig, err
	})

	// check updating the config
	update := config.ShallowClone()
	expected := config.ShallowClone()
	update.Node.BatchPoster.MaxSize += 100
	expected.Node.BatchPoster.MaxSize += 100
	Require(t, liveConfig.Set(update))
	if !reflect.DeepEqual(liveConfig.Get(), expected) {
		Fail(t, "failed to set config")
	}

	// check that an invalid reload gets rejected
	update = config.ShallowClone()
	update.ParentChain.ID++
	if liveConfig.Set(update) == nil {
		Fail(t, "failed to reject invalid update")
	}
	if !reflect.DeepEqual(liveConfig.Get(), expected) {
		Fail(t, "config should not change if its update fails")
	}

	// starting the LiveConfig after testing LiveConfig.set to avoid race condition in the test
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
	if !reflect.DeepEqual(liveConfig.Get(), expected) {
		Fail(t, "live config differs from expected")
	}

	// change the config file
	expected = config.ShallowClone()
	expected.Node.BatchPoster.MaxSize += 100
	jsonConfig = fmt.Sprintf("{\"node\":{\"batch-poster\":{\"max-size\":\"%d\"}}, \"chain\":{\"id\":421613}}", expected.Node.BatchPoster.MaxSize)
	Require(t, WriteToConfigFile(configFile, jsonConfig))

	// trigger LiveConfig reload
	Require(t, syscall.Kill(syscall.Getpid(), syscall.SIGUSR1))

	if !PollLiveConfigUntilEqual(liveConfig, expected) {
		Fail(t, "failed to update config", config.Node.BatchPoster.MaxSize, update.Node.BatchPoster.MaxSize)
	}

	// change chain.id in the config file (currently non-reloadable)
	jsonConfig = fmt.Sprintf("{\"node\":{\"batch-poster\":{\"max-size\":\"%d\"}}, \"chain\":{\"id\":421703}}", expected.Node.BatchPoster.MaxSize)
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

	args := strings.Split("--persistent.chain /tmp/data --init.dev-init --node.parent-chain-reader.enable=false --parent-chain.id 5 --chain.id 421613 --node.batch-poster.parent-chain-wallet.pathname /l1keystore --node.batch-poster.parent-chain-wallet.password passphrase --http.addr 0.0.0.0 --ws.addr 0.0.0.0 --node.sequencer --execution.sequencer.enable --node.feed.output.enable --node.feed.output.port 9642 --node.transaction-streamer.track-block-metadata-from=10", " ")
	args = append(args, []string{"--conf.file", configFile}...)
	config, _, err := ParseNode(context.Background(), args)
	Require(t, err)

	liveConfig := genericconf.NewLiveConfig[*NodeConfig](args, config, func(ctx context.Context, args []string) (*NodeConfig, error) {
		nodeConfig, _, err := ParseNode(ctx, args)
		return nodeConfig, err
	})
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
	if reflect.DeepEqual(liveConfig.Get(), expected) {
		Fail(t, "failed to disable periodic reload")
	}
}

func WriteToConfigFile(path string, jsonConfig string) error {
	return os.WriteFile(path, []byte(jsonConfig), 0600)
}

func PollLiveConfigUntilEqual(liveConfig *genericconf.LiveConfig[*NodeConfig], expected *NodeConfig) bool {
	return PollLiveConfig(liveConfig, expected, true)
}
func PollLiveConfigUntilNotEqual(liveConfig *genericconf.LiveConfig[*NodeConfig], expected *NodeConfig) bool {
	return PollLiveConfig(liveConfig, expected, false)
}

func PollLiveConfig(liveConfig *genericconf.LiveConfig[*NodeConfig], expected *NodeConfig, equal bool) bool {
	for i := 0; i < 16; i++ {
		if reflect.DeepEqual(liveConfig.Get(), expected) == equal {
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
