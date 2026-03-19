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

	cfg := nitroversionalerter.DefaultClientConfig
	cfg.Connection.URL = builder.L2.Stack.HTTPEndpoint()
	connection := rpcclient.NewRpcClient(func() *rpcclient.ClientConfig { return &cfg.Connection }, nil)
	Require(t, connection.Start(ctx))
	alerter := &nitroversionalerter.Client{
		Cfg:        &cfg,
		Connection: connection,
	}

	logHandler.Clear()
	// When our node is above required minimum version, we shouldn't log anything
	alerter.NodeVersion = "v3.2.2"
	nodeVersionTime, err := time.Parse(time.RFC3339, reqNodeVersionDate)
	Require(t, err)
	alerter.NodeVersionTime = nodeVersionTime
	alerter.LogUpgradeMsgIfNecessary(ctx)
	if logHandler.WasLogged(msg) {
		t.Fatal("minimum required node version message should not be logged for correct versioned nodes")
	}

	logHandler.Clear()
	// Node version (v3.2.2) meets requirement, but node version date is shifted 1 minute before
	// the required minimum date, triggering the date check. With default UpgradeGracePeriod (0),
	// now + 0 <= deadline (1h ahead), so INFO level.
	alerter.NodeVersionTime = alerter.NodeVersionTime.Add(-time.Minute)
	alerter.LogUpgradeMsgIfNecessary(ctx)
	if !logHandler.WasLoggedAtLevel(msg, slog.LevelInfo) {
		t.Fatal("minimum required node version message was not logged at level Info")
	}
	if logHandler.WasLoggedAtLevel(msg, slog.LevelWarn) || logHandler.WasLoggedAtLevel(msg, slog.LevelError) {
		t.Fatal("minimum required node version message should only be logged at level Info")
	}

	logHandler.Clear()
	// Node version date equals the required minimum (passes the strict less-than check), but
	// node version "v3.2" is below required "v3.2.1". UpgradeGracePeriod is set large enough
	// that now + gracePeriod > deadline, but now < deadline, so we should see a WARN log.
	alerter.NodeVersionTime = nodeVersionTime
	alerter.NodeVersion = "v3.2"
	alerter.Cfg.UpgradeGracePeriod = 2 * time.Hour
	alerter.LogUpgradeMsgIfNecessary(ctx)
	if !logHandler.WasLoggedAtLevel(msg, slog.LevelWarn) {
		t.Fatal("minimum required node version message was not logged at level Warn")
	}
	if logHandler.WasLoggedAtLevel(msg, slog.LevelInfo) || logHandler.WasLoggedAtLevel(msg, slog.LevelError) {
		t.Fatal("minimum required node version message should only be logged at level Warn")
	}

	logHandler.Clear()
	// Same case as above where node version is still below required minimum.
	// Set upgrade deadline to the past so now exceeds it, should see an ERROR log.
	alerter.Cfg.UpgradeGracePeriod = 0
	builder.nodeConfig.VersionAlerterServer.UpgradeDeadline = time.Now().Add(-time.Minute).Format(time.RFC3339)
	builder.L2.ConsensusConfigFetcher.Set(builder.nodeConfig)
	alerter.LogUpgradeMsgIfNecessary(ctx)
	if !logHandler.WasLoggedAtLevel(msg, slog.LevelError) {
		t.Fatal("minimum required node version message was not logged at level Error")
	}
	if logHandler.WasLoggedAtLevel(msg, slog.LevelInfo) || logHandler.WasLoggedAtLevel(msg, slog.LevelWarn) {
		t.Fatal("minimum required node version message should only be logged at level Error")
	}
}
