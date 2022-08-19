// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package main

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"syscall"
	"testing"

	"github.com/offchainlabs/nitro/util/colors"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func TestSeqConfig(t *testing.T) {
	args := strings.Split("--persistent.chain /tmp/data --init.dev-init --node.l1-reader.enable=false --l1.chain-id 5 --l2.chain-id 421613 --l1.wallet.pathname /l1keystore --l1.wallet.password passphrase --http.addr 0.0.0.0 --ws.addr 0.0.0.0 --node.sequencer.enable --node.feed.output.enable --node.feed.output.port 9642", " ")
	_, _, _, _, _, err := ParseNode(context.Background(), args)
	Require(t, err)
}

func TestUnsafeStakerConfig(t *testing.T) {
	args := strings.Split("--persistent.chain /tmp/data --init.dev-init --node.l1-reader.enable=false --l1.chain-id 5 --l2.chain-id 421613 --l1.wallet.pathname /l1keystore --l1.wallet.password passphrase --http.addr 0.0.0.0 --ws.addr 0.0.0.0 --node.validator.enable --node.validator.strategy MakeNodes --node.validator.staker-interval 10s --node.forwarding-target null --node.validator.dangerous.without-block-validator", " ")
	_, _, _, _, _, err := ParseNode(context.Background(), args)
	Require(t, err)
}

func TestValidatorConfig(t *testing.T) {
	args := strings.Split("--persistent.chain /tmp/data --init.dev-init --node.l1-reader.enable=false --l1.chain-id 5 --l2.chain-id 421613 --l1.wallet.pathname /l1keystore --l1.wallet.password passphrase --http.addr 0.0.0.0 --ws.addr 0.0.0.0 --node.validator.enable --node.validator.strategy MakeNodes --node.validator.staker-interval 10s --node.forwarding-target null", " ")
	_, _, _, _, _, err := ParseNode(context.Background(), args)
	Require(t, err)
}

func TestAggregatorConfig(t *testing.T) {
	args := strings.Split("--persistent.chain /tmp/data --init.dev-init --node.l1-reader.enable=false --l1.chain-id 5 --l2.chain-id 421613 --l1.wallet.pathname /l1keystore --l1.wallet.password passphrase --http.addr 0.0.0.0 --ws.addr 0.0.0.0 --node.sequencer.enable --node.feed.output.enable --node.feed.output.port 9642 --node.data-availability.enable --node.data-availability.rpc-aggregator.backends {[\"url\":\"http://localhost:8547\",\"pubkey\":\"abc==\",\"signerMask\":0x1]}", " ")
	_, _, _, _, _, err := ParseNode(context.Background(), args)
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
	update.Node.Sequencer.ForwardTimeout++
	testUnsafe()
}

func TestLiveNodeConfig(t *testing.T) {
	args := strings.Split("--persistent.chain /tmp/data --init.dev-init --node.l1-reader.enable=false --l1.chain-id 5 --l2.chain-id 421613 --l1.wallet.pathname /l1keystore --l1.wallet.password passphrase --http.addr 0.0.0.0 --ws.addr 0.0.0.0 --node.sequencer.enable --node.feed.output.enable --node.feed.output.port 9642", " ")
	config, _, _, _, _, err := ParseNode(context.Background(), args)
	Require(t, err)
	update, _, _, _, _, err := ParseNode(context.Background(), args)
	Require(t, err)
	update.Node.Sequencer.MaxBlockSpeed++

	liveConfig := NewLiveNodeConfig(args, config)
	if !reflect.DeepEqual(liveConfig.get(), config) {
		Fail(t, "failed to get live config")
	}
	Require(t, liveConfig.set(update))
	if !reflect.DeepEqual(liveConfig.get(), update) {
		Fail(t, "failed to set config")
	}
	// swap pointers, as update is now the config stored in LiveConfig
	config, update = update, config
	// make update not valid for hot reload
	update.L2.ChainID++
	if liveConfig.set(update) == nil {
		Fail(t, "didn't fail when setting a config that is not hot reloadable")
	}
	if !reflect.DeepEqual(liveConfig.get(), config) {
		Fail(t, "config should not change if its update fails")
	}
	// make update valid again
	update.L2.ChainID--
	// sync update to config
	update.Node.Sequencer.MaxBlockSpeed = config.Node.Sequencer.MaxBlockSpeed
	// rename for clarity
	expected := update
	if !reflect.DeepEqual(config, expected) {
		Fail(t, "internal test failure, expected is not in sync with config")
	}
	liveConfig.Start(context.Background())
	if !reflect.DeepEqual(liveConfig.get(), expected) {
		Fail(t, "live config differs from expected")
	}
	err = syscall.Kill(syscall.Getpid(), syscall.SIGUSR1)
	Require(t, err)
	if !reflect.DeepEqual(liveConfig.get(), expected) {
		Fail(t, "live config differs from expected")
	}
	// modifying args won't happen in production, but makes the test easier as there's no need for temporary test config files
	expected.Node.Sequencer.MaxBlockSpeed += 10
	liveConfig.args = append(liveConfig.args, fmt.Sprintf("--node.sequencer.max-block-speed \"%s\"", expected.Node.Sequencer.MaxBlockSpeed.String()))
	// triggering LiveConfig reload, which should overwrite max-block-speed from args
	err = syscall.Kill(syscall.Getpid(), syscall.SIGUSR1)
	// FIXME the reload is not triggered for some reason
	Require(t, err)
	if liveConfig.get() != config {
		Fail(t, "internal test failure, LiveNodeConfig stores unexpected config pointer")
	}
	if !reflect.DeepEqual(config, expected) {
		Fail(t, "config differs from expected")
	}
}

func Require(t *testing.T, err error, text ...interface{}) {
	t.Helper()
	testhelpers.RequireImpl(t, err, text...)
}

func Fail(t *testing.T, printables ...interface{}) {
	t.Helper()
	testhelpers.FailImpl(t, printables...)
}
