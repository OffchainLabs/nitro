// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package arbtest

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"

	nitroversionalerter "github.com/offchainlabs/nitro/arbnode/nitro-version-alerter"
	"github.com/offchainlabs/nitro/nitroversion"
	"github.com/offchainlabs/nitro/util/rpcclient"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func TestNitroNodeVersionAlerter(t *testing.T) {
	logHandler := testhelpers.InitTestLog(t, log.LevelInfo)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reqNodeVersion := "v3.2.1"
	reqNodeVersionTime := time.Now()
	reqNodeVersionDate := reqNodeVersionTime.Format(time.RFC3339)
	upgradeDeadline := time.Now().Add(time.Hour).Format(time.RFC3339)
	msg := "Node version or date is below the minimum requirement, please upgrade"

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false).DontParalellise()
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
	makeVersion := func(tag string, commitTime time.Time) nitroversion.Version {
		version, err := nitroversion.New(nitroversion.Provenance{
			Tag:       tag,
			Commit:    "26b4b9b",
			Timestamp: commitTime.Format(time.RFC3339),
		})
		Require(t, err)
		return version
	}
	alerter := &nitroversionalerter.Client{
		Cfg:        &cfg,
		Connection: connection,
	}

	logHandler.Clear()
	// A newer semantic version and commit time satisfy both requirements.
	alerter.NodeVersion = makeVersion("v3.2.2", reqNodeVersionTime.Add(time.Minute))
	alerter.LogUpgradeMsgIfNecessary(ctx)
	if logHandler.WasLogged(msg) {
		t.Fatal("minimum required node version message should not be logged for correct versioned nodes")
	}

	logHandler.Clear()
	// Equal semantic version and commit time satisfy both requirements.
	alerter.NodeVersion = makeVersion(reqNodeVersion, reqNodeVersionTime)
	alerter.LogUpgradeMsgIfNecessary(ctx)
	if logHandler.WasLogged(msg) {
		t.Fatal("minimum required node version message should not be logged for equal versioned nodes")
	}

	logHandler.Clear()
	// A version-only requirement does not depend on the local commit timestamp.
	builder.nodeConfig.VersionAlerterServer.MinRequiredNitroByDate = ""
	builder.L2.ConsensusConfigFetcher.Set(builder.nodeConfig)
	alerter.NodeVersion = makeVersion("v3.2.2", reqNodeVersionTime.Add(-time.Minute))
	alerter.LogUpgradeMsgIfNecessary(ctx)
	if logHandler.WasLogged(msg) {
		t.Fatal("minimum required node version message should not be logged when a version-only requirement is met")
	}

	logHandler.Clear()
	// A date-only requirement uses the typed commit-time comparison without
	// trying to parse an absent semantic version.
	builder.nodeConfig.VersionAlerterServer.MinRequiredNitroByVersion = ""
	builder.nodeConfig.VersionAlerterServer.MinRequiredNitroByDate = reqNodeVersionDate
	builder.L2.ConsensusConfigFetcher.Set(builder.nodeConfig)
	alerter.LogUpgradeMsgIfNecessary(ctx)
	if !logHandler.WasLoggedAtLevel(msg, slog.LevelInfo) {
		t.Fatal("minimum required node version message was not logged at level Info")
	}
	if logHandler.WasLoggedAtLevel(msg, slog.LevelWarn) || logHandler.WasLoggedAtLevel(msg, slog.LevelError) {
		t.Fatal("minimum required node version message should only be logged at level Info")
	}

	logHandler.Clear()
	// Node version date equals the required minimum (passes the strict less-than check), but
	// node version "v3.2.0" is below required "v3.2.1". UpgradeGracePeriod is set large enough
	// that now + gracePeriod > deadline, but now < deadline, so we should see a WARN log.
	builder.nodeConfig.VersionAlerterServer.MinRequiredNitroByVersion = reqNodeVersion
	builder.L2.ConsensusConfigFetcher.Set(builder.nodeConfig)
	alerter.NodeVersion = makeVersion("v3.2.0", reqNodeVersionTime)
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
