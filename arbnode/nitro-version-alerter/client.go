package nitroversionalerter

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/pflag"
	"golang.org/x/mod/semver"

	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/bold/util/stopwaiter"
	"github.com/offchainlabs/nitro/cmd/util/confighelpers"
	"github.com/offchainlabs/nitro/util/rpcclient"
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
	Cfg             *ClientConfig
	Connection      *rpcclient.RpcClient
	NodeVersion     string
	NodeVersionTime time.Time
}

func NewClient(ctx context.Context, cfg *ClientConfig) (*Client, error) {
	nodeVersion, _, nodeVersionDate := confighelpers.GetVersion()
	if !semver.IsValid(nodeVersion) {
		log.Warn("asfdaf", "nodeVersion", nodeVersion, "nodeVersionDate", nodeVersionDate)
		return nil, nil
	}
	nodeVersionTime, err := time.Parse(time.RFC3339, nodeVersionDate)
	if err != nil {
		return nil, fmt.Errorf("error parsing nodeVersionDate: %s into time: %w", nodeVersionDate, err)
	}
	connectionConfigFetcher := func() *rpcclient.ClientConfig { return &cfg.Connection }
	connection := rpcclient.NewRpcClient(connectionConfigFetcher, nil)
	if err = connection.Start(ctx); err != nil {
		return nil, err
	}
	return &Client{
		Cfg:             cfg,
		Connection:      connection,
		NodeVersion:     nodeVersion,
		NodeVersionTime: nodeVersionTime,
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
	if res.NodeVersion != "" && semver.Compare(c.NodeVersion, res.NodeVersion) < 0 { // node is not up to date
		needLogging = true
	}
	if res.NodeVersionDate != "" {
		minRequiredVersionTime, err := time.Parse(time.RFC3339, res.NodeVersionDate)
		if err != nil {
			log.Error("Cannot parse NodeVersionDate returned by arb_getMinRequiredNitroVersion into time", "err", err)
			return c.Cfg.PingInterval
		}
		if c.NodeVersionTime.Compare(minRequiredVersionTime) < 0 { // node is not up to date
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
		"currentNodeVersion", c.NodeVersion, "currentNodeVersionDate", c.NodeVersionTime,
	)
	return c.Cfg.PingInterval
}
