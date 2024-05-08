package arbos

import (
	"encoding/json"
	"reflect"
	"testing"

	tagged_base64 "github.com/EspressoSystems/espresso-sequencer-go/tagged-base64"
	espressoTypes "github.com/EspressoSystems/espresso-sequencer-go/types"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
)

func TestEspressoParsing(t *testing.T) {
	expectTxes := []espressoTypes.Bytes{
		[]byte{1, 2, 3},
		[]byte{4},
	}
	expectHeader := &arbostypes.L1IncomingMessageHeader{
		Kind:        arbostypes.L1MessageType_L2Message,
		BlockNumber: 1,
	}
	var mockProof = json.RawMessage(`{"NonExistence":{"ns_id":0}}`)
	var mockChainConfig = &espressoTypes.ResolvableChainConfig{
		ChainConfig: espressoTypes.EitherChainConfig{
			Left: &espressoTypes.ChainConfig{ChainId: *espressoTypes.NewU256().SetUint64(0x8a19), MaxBlockSize: 10240, BaseFee: *espressoTypes.NewU256().SetUint64(0)},
		},
	}
	mockCommitment, err := tagged_base64.New("payloadCommitment", []byte{1, 2, 3})
	Require(t, err)
	root, err := tagged_base64.New("root", []byte{4, 5, 6})
	Require(t, err)
	expectJst := &arbostypes.EspressoBlockJustification{
		Header: espressoTypes.Header{
			L1Head:              1,
			ChainConfig:         mockChainConfig,
			Timestamp:           2,
			Height:              3,
			NsTable:             &espressoTypes.NsTable{Bytes: []byte{1}},
			L1Finalized:         &espressoTypes.L1BlockInfo{},
			PayloadCommitment:   mockCommitment,
			BuilderCommitment:   mockCommitment,
			BlockMerkleTreeRoot: root,
			FeeMerkleTreeRoot:   root,
			FeeInfo:             &espressoTypes.FeeInfo{},
		},
		Proof: &mockProof,
	}
	msg, err := MessageFromEspresso(expectHeader, expectTxes, expectJst)
	Require(t, err)

	actualTxes, actualJst, err := ParseEspressoMsg(&msg)
	Require(t, err)

	if !reflect.DeepEqual(actualTxes, expectTxes) {
		Fail(t)
	}

	if !reflect.DeepEqual(actualJst.Header, expectJst.Header) {
		Fail(t)
	}
}
