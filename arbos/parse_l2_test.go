package arbos

import (
	"reflect"
	"testing"

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
	expectJst := &arbostypes.EspressoBlockJustification{
		Header: espressoTypes.Header{
			TransactionsRoot: espressoTypes.NmtRoot{Root: []byte{7, 8, 9}},
			L1Head:           1,
			Timestamp:        2,
			Height:           3,
			L1Finalized:      &espressoTypes.L1BlockInfo{},
		},
		Proof: []byte{9},
	}
	msg, err := MessageFromEspresso(expectHeader, expectTxes, expectJst)
	Require(t, err)

	actualTxes, actualJst, err := ParseEspressoMsg(&msg)
	Require(t, err)

	if !reflect.DeepEqual(actualTxes, expectTxes) {
		Fail(t)
	}

	if !reflect.DeepEqual(actualJst.Proof, expectJst.Proof) {
		Fail(t)
	}
	if !reflect.DeepEqual(actualJst.Header.TransactionsRoot, expectJst.Header.TransactionsRoot) {
		Fail(t)
	}

}
