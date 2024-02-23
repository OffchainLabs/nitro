package arbos

import (
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
	payloadCommitment, err := tagged_base64.New("payloadCommitment", []byte{1, 2, 3})
	Require(t, err)
	root, err := tagged_base64.New("root", []byte{4, 5, 6})
	Require(t, err)
	expectJst := &arbostypes.EspressoBlockJustification{
		Header: espressoTypes.Header{
			L1Head:              1,
			Timestamp:           2,
			Height:              3,
			NsTable:             &espressoTypes.NsTable{RawPayload: []byte{1}},
			L1Finalized:         &espressoTypes.L1BlockInfo{},
			PayloadCommitment:   payloadCommitment,
			BlockMerkleTreeRoot: root,
			FeeMerkleTreeRoot:   root,
			FeeInfo:             &espressoTypes.FeeInfo{},
		},
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
