// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package main

import (
	"context"
	"github.com/offchainlabs/nitro/util/testhelpers"
	"strings"
	"testing"
)

func TestSeqConfig(t *testing.T) {
	args := strings.Split("--persistent.data /data --dev-init --l1.url ws://geth:8546 --l1.wallet.pathname /l1keystore --l1.wallet.password passphrase --l1.deployment /deploydata/deployment.json --http.addr 0.0.0.0 --ws.addr 0.0.0.0 --node.sequencer.enable --node.feed.output.enable --node.feed.output.port 9642", " ")
	_, _, _, err := ParseNode(context.Background(), args)
	testhelpers.RequireImpl(t, err)
}

func TestUnsafeStakerConfig(t *testing.T) {
	args := strings.Split("--persistent.data /data --dev-init --l1.url ws://geth:8546 --l1.wallet.pathname /l1keystore --l1.wallet.password passphrase --l1.deployment /deploydata/deployment.json --http.addr 0.0.0.0 --ws.addr 0.0.0.0 --node.validator.enable --node.validator.strategy MakeNodes --node.validator.staker-interval 10s --node.forwarding-target null --node.validator.dangerous.without-block-validator", " ")
	_, _, _, err := ParseNode(context.Background(), args)
	testhelpers.RequireImpl(t, err)
}

func TestValidatorConfig(t *testing.T) {
	args := strings.Split("--persistent.data /data --dev-init --l1.url ws://geth:8546 --l1.wallet.pathname /l1keystore --l1.wallet.password passphrase --l1.deployment /deploydata/deployment.json --http.addr 0.0.0.0 --ws.addr 0.0.0.0 --node.validator.enable --node.validator.strategy MakeNodes --node.validator.staker-interval 10s --node.forwarding-target null", " ")
	_, _, _, err := ParseNode(context.Background(), args)
	testhelpers.RequireImpl(t, err)
}
