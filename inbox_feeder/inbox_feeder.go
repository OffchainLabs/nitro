package inbox_feeder

import (
	"context"
	"encoding/binary"
	"github.com/OffchainLabs/new-rollup-exploration/protocol"
	"github.com/ethereum/go-ethereum/crypto"
	"time"
)

func StartInboxFeeder(ctx context.Context, chain *protocol.AssertionChain, messageInterval time.Duration, randomSeed []byte) {
	go func() {
		ticker := chain.TimeReference().NewTicker(messageInterval)
		defer ticker.Stop()
		msgNum := uint64(0)
		for {
			select {
			case <-ticker.C():
				message := crypto.Keccak256(binary.BigEndian.AppendUint64(randomSeed, msgNum))
				_ = chain.Tx(func(tx *protocol.ActiveTx, innerChain *protocol.AssertionChain) error {
					innerChain.Inbox().Append(tx, message)
					return nil
				})
			case <-ctx.Done():
				return
			}
		}
	}()
}
