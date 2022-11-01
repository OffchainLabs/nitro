package arbnode

import (
	"bytes"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
)

var (
	Uint256, _ = abi.NewType("uint256", "", nil)
	Address, _ = abi.NewType("address", "", nil)
)

func Test_encodeAddBatch(t *testing.T) {
	seqNum := big.NewInt(1)
	prevMsgNum := arbutil.MessageIndex(2)
	newMsgNum := arbutil.MessageIndex(3)
	l2MessageDataRaw := "foobar"
	l2MessageData := []byte(l2MessageDataRaw)
	delayedMsg := uint64(4)
	seqInboxABI, err := bridgegen.SequencerInboxMetaData.GetAbi()
	if err != nil {
		t.Fatal(err)
	}
	seqInboxABI.Methods[sequencerBatchPostWithBlobsMethodName] = abi.NewMethod(
		sequencerBatchPostWithBlobsMethodName,
		sequencerBatchPostWithBlobsMethodName,
		abi.Function,
		"",    // Mutability
		false, // isConst
		false, // isPayable
		[]abi.Argument{
			{"sequenceNumber", Uint256, false},
			{"afterDelayedMessagesRead", Uint256, false},
			{"gasRefunder", Address, false},
			{"prevMessageCount", Uint256, false},
			{"newMessageCount", Uint256, false},
		},
		nil, // outputs
	)
	t.Run("eip-4844 mode separates L2 message data from calldata", func(t *testing.T) {
		b := &BatchPoster{
			seqInboxABI: seqInboxABI,
			config: func() *BatchPosterConfig {
				return &BatchPosterConfig{EIP4844: true}
			},
		}
		result, err := b.encodeAddBatch(
			seqNum, prevMsgNum, newMsgNum, l2MessageData, delayedMsg,
		)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(result.L2MessageData, l2MessageData) {
			t.Error("Expected L2 message to be provided separately from calldata when EIP-4844")
		}
		if strings.Contains(string(result.SequencerInboxCalldata), "foobar") {
			t.Error("Expected calldata to not embed L2 message data when EIP-4844")
		}
	})
	t.Run("non-eip-4844 mode preserves L2 message data within the tx calldata", func(t *testing.T) {
		b := &BatchPoster{
			seqInboxABI: seqInboxABI,
			config: func() *BatchPosterConfig {
				return &BatchPosterConfig{EIP4844: false}
			},
		}
		result, err := b.encodeAddBatch(
			seqNum, prevMsgNum, newMsgNum, l2MessageData, delayedMsg,
		)
		if err != nil {
			t.Fatal(err)
		}
		if len(result.L2MessageData) > 0 {
			t.Fatal("Expected empty L2 message data outside of calldata when not EIP-4844")
		}
		if !strings.Contains(string(result.SequencerInboxCalldata), "foobar") {
			t.Error("Expected calldata to embed L2 message data when not EIP-4844")
		}
	})
}
