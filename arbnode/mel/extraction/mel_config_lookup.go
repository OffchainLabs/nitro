// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package melextraction

import (
	"context"

	"github.com/ethereum/go-ethereum/core/types"

	"github.com/offchainlabs/nitro/arbnode/mel"
	"github.com/offchainlabs/nitro/solgen/go/rollupgen"
)

// ParseMELConfigFromBlock scans the logs of the given parent chain block for
// a MELConfigEvent. The log prefetcher already filters by rollup address,
// so this function only needs to match the event topic.
// Returns nil if no config event is found in the block.
func ParseMELConfigFromBlock(
	ctx context.Context,
	parentChainHeader *types.Header,
	logsFetcher LogsFetcher,
	eventUnpacker EventUnpacker,
) (*mel.MELConfig, error) {
	logs, err := logsFetcher.LogsForBlockHash(ctx, parentChainHeader.Hash())
	if err != nil {
		return nil, err
	}
	for _, log := range logs {
		if log == nil || len(log.Topics) == 0 || log.Topics[0] != MELConfigEventID {
			continue
		}
		event := new(rollupgen.RollupAdminLogicMELConfigSet)
		if err := eventUnpacker.UnpackLogTo(event, RollupAdminABI, "MELConfigSet", *log); err != nil {
			return nil, err
		}
		return &mel.MELConfig{
			Inbox:          event.Inbox,
			SequencerInbox: event.SequencerInbox,
		}, nil
	}
	return nil, nil
}
