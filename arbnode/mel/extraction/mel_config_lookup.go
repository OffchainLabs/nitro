// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package melextraction

import (
	"context"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/offchainlabs/nitro/arbnode/mel"
)

// melConfigEventABI is a manually defined ABI for the MELConfigEvent.
// This can be replaced with rollupgen.RollupAdminLogicMetaData.GetAbi()
// once the Go bindings are regenerated from the updated Solidity contracts.
var melConfigEventABI *abi.ABI

func init() {
	const melConfigEventABIJSON = `[{"anonymous":false,"inputs":[{"indexed":false,"internalType":"address","name":"inbox","type":"address"},{"indexed":false,"internalType":"address","name":"sequencerInbox","type":"address"},{"indexed":false,"internalType":"uint256","name":"melVersionActivationBlock","type":"uint256"}],"name":"MELConfigEvent","type":"event"}]`
	parsed, err := abi.JSON(strings.NewReader(melConfigEventABIJSON))
	if err != nil {
		panic(err)
	}
	melConfigEventABI = &parsed
}

// melConfigEventFields holds the decoded fields from a MELConfigEvent log.
type melConfigEventFields struct {
	Inbox                     common.Address
	SequencerInbox            common.Address
	MelVersionActivationBlock *big.Int
}

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
		event := new(melConfigEventFields)
		if err := eventUnpacker.UnpackLogTo(event, melConfigEventABI, "MELConfigEvent", *log); err != nil {
			return nil, err
		}
		if !event.MelVersionActivationBlock.IsUint64() {
			return nil, nil
		}
		return &mel.MELConfig{
			Inbox:                  event.Inbox,
			SequencerInbox:         event.SequencerInbox,
			VersionActivationBlock: event.MelVersionActivationBlock.Uint64(),
		}, nil
	}
	return nil, nil
}
