// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package main

import (
	"context"
	"fmt"
	"reflect"
	"strings"
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

func Require(t *testing.T, err error, text ...interface{}) {
	t.Helper()
	testhelpers.RequireImpl(t, err, text...)
}

func Fail(t *testing.T, printables ...interface{}) {
	t.Helper()
	testhelpers.FailImpl(t, printables...)
}
