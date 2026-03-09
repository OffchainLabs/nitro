// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package timeboost

import (
	"time"

	"github.com/spf13/pflag"
)

type Config struct {
	Enable                       bool          `koanf:"enable"`
	AuctionContractAddress       string        `koanf:"auction-contract-address"`
	AuctioneerAddress            string        `koanf:"auctioneer-address"`
	ExpressLaneAdvantage         time.Duration `koanf:"express-lane-advantage"`
	SequencerHTTPEndpoint        string        `koanf:"sequencer-http-endpoint"`
	EarlySubmissionGrace         time.Duration `koanf:"early-submission-grace"`
	MaxFutureSequenceDistance    uint64        `koanf:"max-future-sequence-distance"`
	RedisUrl                     string        `koanf:"redis-url"`
	RedisUpdateEventsChannelSize uint64        `koanf:"redis-update-events-channel-size"`
	QueueTimeoutInBlocks         uint64        `koanf:"queue-timeout-in-blocks"`
}

var DefaultConfig = Config{
	Enable:                       false,
	AuctionContractAddress:       "",
	AuctioneerAddress:            "",
	ExpressLaneAdvantage:         time.Millisecond * 200,
	SequencerHTTPEndpoint:        "http://localhost:8547",
	EarlySubmissionGrace:         time.Second * 2,
	MaxFutureSequenceDistance:    1000,
	RedisUrl:                     "unset",
	RedisUpdateEventsChannelSize: 500,
	QueueTimeoutInBlocks:         5,
}

func AddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".enable", DefaultConfig.Enable, "enable timeboost based on express lane auctions")
	f.String(prefix+".auction-contract-address", DefaultConfig.AuctionContractAddress, "Address of the proxy pointing to the ExpressLaneAuction contract")
	f.String(prefix+".auctioneer-address", DefaultConfig.AuctioneerAddress, "Address of the Timeboost Autonomous Auctioneer")
	f.Duration(prefix+".express-lane-advantage", DefaultConfig.ExpressLaneAdvantage, "specify the express lane advantage")
	f.String(prefix+".sequencer-http-endpoint", DefaultConfig.SequencerHTTPEndpoint, "this sequencer's http endpoint")
	f.Duration(prefix+".early-submission-grace", DefaultConfig.EarlySubmissionGrace, "period of time before the next round where submissions for the next round will be queued")
	f.Uint64(prefix+".max-future-sequence-distance", DefaultConfig.MaxFutureSequenceDistance, "maximum allowed difference (in terms of sequence numbers) between a future express lane tx and the current sequence count of a round")
	f.String(prefix+".redis-url", DefaultConfig.RedisUrl, "the Redis URL for expressLaneService to coordinate via")
	f.Uint64(prefix+".redis-update-events-channel-size", DefaultConfig.RedisUpdateEventsChannelSize, "size of update events' buffered channels in timeboost redis coordinator")
	f.Uint64(prefix+".queue-timeout-in-blocks", DefaultConfig.QueueTimeoutInBlocks, "maximum amount of time (measured in blocks) that Express Lane transactions can wait in the sequencer's queue")
}
