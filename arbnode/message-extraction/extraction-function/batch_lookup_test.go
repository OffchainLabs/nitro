package extractionfunction

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/stretchr/testify/require"
)

func Test_parseBatchesFromBlock(t *testing.T) {
	event := &bridgegen.SequencerInboxSequencerBatchDelivered{
		BatchSequenceNumber:      big.NewInt(1),
		BeforeAcc:                common.BytesToHash([]byte{1}),
		AfterAcc:                 common.BytesToHash([]byte{2}),
		DelayedAcc:               common.BytesToHash([]byte{3}),
		AfterDelayedMessagesRead: big.NewInt(4),
		TimeBounds: bridgegen.IBridgeTimeBounds{
			MinTimestamp:   0,
			MaxTimestamp:   100,
			MinBlockNumber: 0,
			MaxBlockNumber: 100,
		},
		DataLocation: 1,
	}
	eventABI := seqInboxABI.Events["SequencerBatchDelivered"]
	packedLog, err := eventABI.Inputs.Pack(
		event.BatchSequenceNumber,
		event.BeforeAcc,
		event.AfterAcc,
		event.DelayedAcc,
		event.AfterDelayedMessagesRead,
		event.TimeBounds,
		event.DataLocation,
	)
	require.NoError(t, err)
	_ = packedLog
	addr := common.HexToAddress("0x1234567890123456789012345678901234567890")
	blockHeader := &types.Header{}
	txData := &types.DynamicFeeTx{
		To:        &addr,
		Nonce:     1,
		GasFeeCap: big.NewInt(1),
		GasTipCap: big.NewInt(1),
		Gas:       1,
		Value:     big.NewInt(0),
		Data:      nil,
	}
	tx := types.NewTx(txData)
	blockBody := &types.Body{
		Transactions: []*types.Transaction{tx},
	}
	block := types.NewBlock(
		blockHeader,
		blockBody,
		nil,
		nil,
	)
	_ = block
}
