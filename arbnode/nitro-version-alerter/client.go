// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package nitroversionalerter

import (
	"context"
	"time"

	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/nitroversion"
	"github.com/offchainlabs/nitro/util/rpcclient"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

type ClientConfig struct {
	Enable             bool                   `koanf:"enable"`
	Connection         rpcclient.ClientConfig `koanf:"connection" reload:"hot"`
	UpgradeGracePeriod time.Duration          `koanf:"upgrade-grace-period"`
	PingInterval       time.Duration          `koanf:"ping-interval"`
}

func (c *ClientConfig) Validate() error {
	if !c.Enable {
		return nil
	}
	return c.Connection.Validate()
}

var DefaultClientConfig = ClientConfig{
	Enable:             false,
	Connection:         rpcclient.DefaultClientConfig,
	UpgradeGracePeriod: 0,
	PingInterval:       5 * time.Minute,
}

func ClientConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".enable", DefaultClientConfig.Enable, "enable querying arb_getMinRequiredNitroVersion endpoint in regular intervals and firing alerts if the node software is below the required version")
	rpcclient.RPCClientAddOptions(prefix+".connection", f, &DefaultClientConfig.Connection)
	f.Duration(prefix+".upgrade-grace-period", DefaultClientConfig.UpgradeGracePeriod, "represents grace period up until the upgrade deadline received from arb_getMinRequiredNitroVersion, determines escalation of messages regarding node software upgrade")
	f.Duration(prefix+".ping-interval", DefaultClientConfig.PingInterval, "how often the nitro version alerter should ping arb_getMinRequiredNitroVersion for data")
}

type Client struct {
	stopwaiter.StopWaiter
	Cfg         *ClientConfig
	Connection  *rpcclient.RpcClient
	NodeVersion nitroversion.Version
}

func NewClient(ctx context.Context, cfg *ClientConfig) (*Client, error) {
	nodeVersion := nitroversion.Current()
	if !nodeVersion.IsSemverTagged() {
		log.Warn("node version is not a semantic-version-tagged release, skipping version alerter", "version", nodeVersion)
		return nil, nil
	}
	connectionConfigFetcher := func() *rpcclient.ClientConfig { return &cfg.Connection }
	connection := rpcclient.NewRpcClient(connectionConfigFetcher, nil)
	if err := connection.Start(ctx); err != nil {
		return nil, err
	}
	return &Client{
		Cfg:         cfg,
		Connection:  connection,
		NodeVersion: nodeVersion,
	}, nil
}

func (c *Client) Start(ctx context.Context) {
	c.CallIteratively(c.LogUpgradeMsgIfNecessary)
}

func (c *Client) LogUpgradeMsgIfNecessary(ctx context.Context) time.Duration {
	var res MinRequiredNitroVersionResult
	err := c.Connection.CallContext(ctx, &res, "arb_getMinRequiredNitroVersion")
	if err != nil {
		log.Error("Fetching upgrade info from arb_getMinRequiredNitroVersion endpoint failed", "err", err)
		return c.Cfg.PingInterval
	}
	if res.UpgradeDeadline == "" || (res.NodeVersion == "" && res.NodeVersionDate == "") {
		return c.Cfg.PingInterval
	}

	var needLogging bool
	if res.NodeVersion != "" {
		targetVersion, err := nitroversion.ParseCanonicalVersion(res.NodeVersion)
		if err != nil {
			log.Error("Cannot parse NodeVersion returned by arb_getMinRequiredNitroVersion into canonical version", "version", res.NodeVersion, "err", err)
			return c.Cfg.PingInterval
		}
		if c.NodeVersion.IsVersionOlderThan(targetVersion) { // node is not up to date
			needLogging = true
		}
	}
	if res.NodeVersionDate != "" {
		minRequiredVersionTime, err := time.Parse(time.RFC3339, res.NodeVersionDate)
		if err != nil {
			log.Error("Cannot parse NodeVersionDate returned by arb_getMinRequiredNitroVersion into time", "timestamp", res.NodeVersionDate, "err", err)
			return c.Cfg.PingInterval
		}
		if c.NodeVersion.IsCommitTimestampOlderThan(minRequiredVersionTime) { // node is not up to date
			needLogging = true
		}
	}
	if !needLogging {
		return c.Cfg.PingInterval
	}
	upgradeDeadline, err := time.Parse(time.RFC3339, res.UpgradeDeadline)
	if err != nil {
		log.Error("Cannot parse UpgradeDeadline returned by arb_getMinRequiredNitroVersion into time", "err", err)
		return c.Cfg.PingInterval
	}
	now := time.Now()
	var logLevel func(string, ...interface{})
	if now.UnixNano() > upgradeDeadline.UnixNano() {
		logLevel = log.Error
	} else if now.UnixNano() < upgradeDeadline.UnixNano() && now.UnixNano()+c.Cfg.UpgradeGracePeriod.Nanoseconds() > upgradeDeadline.UnixNano() {
		logLevel = log.Warn
	} else if now.UnixNano()+c.Cfg.UpgradeGracePeriod.Nanoseconds() <= upgradeDeadline.UnixNano() {
		logLevel = log.Info
	} else {
		log.Error("Could not compare current time and UpgradeGracePeriod with upgradeDeadline", "now", now, "upgradeGracePeriod", c.Cfg.UpgradeGracePeriod, "upgradeDeadline", upgradeDeadline)
		return c.Cfg.PingInterval
	}
	logLevel("Node version or date is below the minimum requirement, please upgrade",
		"requiredVersion", res.NodeVersion, "requiredNodeVersionDate", res.NodeVersionDate, "upgradeDeadline", res.UpgradeDeadline,
		"currentNodeVersion", c.NodeVersion,
	)
	return c.Cfg.PingInterval
}
