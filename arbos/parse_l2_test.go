package arbos

import (
	"reflect"
	"testing"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbos/espresso"
)

func TestEspressoParsing(t *testing.T) {
	expectTxes := []espresso.Bytes{
		[]byte{1, 2, 3},
		[]byte{4},
	}
	expectHeader := &arbostypes.L1IncomingMessageHeader{
		Kind:        arbostypes.L1MessageType_L2Message,
		BlockNumber: 1,
	}
	expectJst := &arbostypes.EspressoBlockJustification{
		Header: espresso.Header{
			TransactionsRoot: espresso.NmtRoot{Root: []byte{7, 8, 9}},
			Metadata: espresso.Metadata{
				L1Head:    1,
				Timestamp: 2,
			},
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

	if !reflect.DeepEqual(actualJst, expectJst) {
		Fail(t)
	}

}
