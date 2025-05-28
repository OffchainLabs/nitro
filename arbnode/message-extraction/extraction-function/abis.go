package extractionfunction

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
)

var batchDeliveredID common.Hash
var seqInboxABI *abi.ABI

func init() {
	var err error
	sequencerBridgeABI, err := bridgegen.SequencerInboxMetaData.GetAbi()
	if err != nil {
		panic(err)
	}
	batchDeliveredID = sequencerBridgeABI.Events["SequencerBatchDelivered"].ID
	seqInboxABI, err = bridgegen.SequencerInboxMetaData.GetAbi()
	if err != nil {
		panic(err)
	}
}
