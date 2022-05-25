package das

import (
	"context"
	"errors"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/util/arbmath"
	"time"
)

func SyncStorageServiceFromChain(
	ctx context.Context,
	syncTo StorageService,
	dataSource arbstate.SimpleDASReader,
	l1client arbutil.L1Interface,
	seqInboxAddr common.Address,
	lowerBoundL1BlockNum *uint64,
	expirationTime uint64,
	stopWhenCaughtUp bool,
) error {
	// make sure that as we sync, any Keysets missing from dataSource will fetched from the L1 chain
	dataSource, err := NewChainFetchSimpleDASReader(dataSource, l1client, seqInboxAddr)
	if err != nil {
		return err
	}

	seqInbox, err := bridgegen.NewSequencerInbox(seqInboxAddr, l1client)
	if err != nil {
		return err
	}
	seqInboxFilterer := seqInbox.SequencerInboxFilterer
	eventChan := make(chan *bridgegen.SequencerInboxSequencerBatchData)
	defer close(eventChan)
	subscription, err := seqInboxFilterer.WatchSequencerBatchData(&bind.WatchOpts{Context: ctx, Start: lowerBoundL1BlockNum}, eventChan, nil)
	if err != nil {
		return err
	}
	defer subscription.Unsubscribe()

	latestL1BlockNumber, err := l1client.BlockNumber(ctx)
	if err != nil {
		return err
	}
	for {
		select {
		case event := <-eventChan:
			data := event.Data
			if len(data) >= 41 && arbstate.IsDASMessageHeaderByte(data[40]) {
				preimages := make(map[common.Hash][]byte)
				if _, err = arbstate.RecoverPayloadFromDasBatch(ctx, data, dataSource, preimages); err != nil {
					return err
				}
				for hash, contents := range preimages {
					_, err := syncTo.GetByHash(ctx, hash.Bytes())
					if errors.Is(err, ErrNotFound) {
						if err := syncTo.Put(ctx, contents, arbmath.SaturatingUAdd(uint64(time.Now().Unix()), expirationTime)); err != nil {
							return err
						}
					} else if err != nil {
						return err
					}
				}
			}
			if stopWhenCaughtUp {
				if event.Raw.BlockNumber >= latestL1BlockNumber {
					return syncTo.Sync(ctx)
				}
				latestL1BlockNumber, err = l1client.BlockNumber(ctx)
				if err != nil {
					return err
				}
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
