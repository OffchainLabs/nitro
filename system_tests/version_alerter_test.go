// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package arbtest

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbnode/nitro-version-alerter"
	"github.com/offchainlabs/nitro/util/rpcclient"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func TestNitroNodeVersionAlerter(t *testing.T) {
	logHandler := testhelpers.InitTestLog(t, log.LevelInfo)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reqNodeVersion := "v3.2.1"
	reqNodeVersionDate := time.Now().Format(time.RFC3339)
	// Use a deadline far in the future to avoid races with Build() startup time
	upgradeDeadline := time.Now().Add(time.Hour).Format(time.RFC3339)
	msg := "Node version or date is below the minimum requirement, please upgrade"

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	builder.nodeConfig.VersionAlerterServer.Enable = true
	builder.nodeConfig.VersionAlerterServer.MinRequiredNitroByVersion = reqNodeVersion
	builder.nodeConfig.VersionAlerterServer.MinRequiredNitroByDate = reqNodeVersionDate
	builder.nodeConfig.VersionAlerterServer.UpgradeDeadline = upgradeDeadline
	builder.l2StackConfig.HTTPHost = "localhost"
	builder.l2StackConfig.HTTPModules = []string{"eth", "arb"}
	cleanup := builder.Build(t)
	defer cleanup()

	l2rpc := builder.L2.Stack.Attach()
	var res nitroversionalerter.MinRequiredNitroVersionResult
	err := l2rpc.CallContext(ctx, &res, "arb_getMinRequiredNitroVersion")
	Require(t, err)
	if res.NodeVersion != reqNodeVersion || res.NodeVersionDate != reqNodeVersionDate || res.UpgradeDeadline != upgradeDeadline {
		t.Fatal("unexpected min required node version, by date or upgrade deadline received from the arb_getMinRequiredNitroVersion rpc")
	}

	cfg := &nitroversionalerter.DefaultClientConfig
	cfg.Connection.URL = builder.L2.Stack.HTTPEndpoint()
	connectionConfigFetcher := func() *rpcclient.ClientConfig { return &cfg.Connection }
	connection := rpcclient.NewRpcClient(connectionConfigFetcher, nil)
	Require(t, connection.Start(ctx))
	alerter := &nitroversionalerter.Client{
		Cfg:        cfg,
		Connection: connection,
	}

	// When our node is above required minimum version, we shouldn't log anything
	alerter.NodeVersion = "v3.2.2"
	nodeVersionTime, err := time.Parse(time.RFC3339, reqNodeVersionDate)
	Require(t, err)
	alerter.NodeVersionTime = nodeVersionTime
	alerter.LogUpgradeMsgIfNecessary(ctx)
	if logHandler.WasLogged(msg) {
		t.Fatal("minimum required node version message should not be logged for correct versioned nodes")
	}

	// Node version is above required minimum version but upgrade time is not.
	// Deadline is far in the future so now + gracePeriod(0) <= deadline → INFO
	alerter.NodeVersionTime = alerter.NodeVersionTime.Add(-1 * time.Minute)
	alerter.LogUpgradeMsgIfNecessary(ctx)
	if !logHandler.WasLoggedAtLevel(msg, slog.LevelInfo) {
		t.Fatal("minimum required node version message was not logged at level Info")
	}

	// Node version is below required minimum. Set deadline to 5s in the future
	// and gracePeriod to 10s so now + gracePeriod > deadline → WARN
	alerter.NodeVersionTime = nodeVersionTime
	alerter.NodeVersion = "v3.2"
	alerter.Cfg.UpgradeGracePeriod = 10 * time.Second
	warnDeadline := time.Now().Add(5 * time.Second).Format(time.RFC3339)
	builder.nodeConfig.VersionAlerterServer.UpgradeDeadline = warnDeadline
	builder.L2.ConsensusConfigFetcher.Set(builder.nodeConfig)
	alerter.LogUpgradeMsgIfNecessary(ctx)
	if !logHandler.WasLoggedAtLevel(msg, slog.LevelWarn) {
		t.Fatal("minimum required node version message was not logged at level Warn")
	}

	// Set deadline to the past so now > deadline → ERROR
	pastDeadline := time.Now().Add(-1 * time.Second).Format(time.RFC3339)
	builder.nodeConfig.VersionAlerterServer.UpgradeDeadline = pastDeadline
	builder.L2.ConsensusConfigFetcher.Set(builder.nodeConfig)
	alerter.LogUpgradeMsgIfNecessary(ctx)
	if !logHandler.WasLoggedAtLevel(msg, slog.LevelError) {
		t.Fatal("minimum required node version message was not logged at level Error")
	}
}
