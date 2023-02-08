package inbox_feeder

import (
	"context"
	"encoding/binary"
	"time"

	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	"github.com/ethereum/go-ethereum/crypto"
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
				_ = chain.Tx(func(tx *protocol.ActiveTx) error {
					chain.Inbox().Append(tx, message)
					return nil
				})
			case <-ctx.Done():
				return
			}
		}
	}()
}
