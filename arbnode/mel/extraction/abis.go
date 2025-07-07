package melextraction

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
)

var batchDeliveredID common.Hash
var inboxMessageDeliveredID common.Hash
var inboxMessageFromOriginID common.Hash
var seqInboxABI *abi.ABI
var iBridgeABI *abi.ABI
var iInboxABI *abi.ABI
var iDelayedMessageProviderABI *abi.ABI

func init() {
	var err error
	sequencerBridgeABI, err := bridgegen.SequencerInboxMetaData.GetAbi()
	if err != nil {
		panic(err)
	}
	batchDeliveredID = sequencerBridgeABI.Events["SequencerBatchDelivered"].ID
	parsedIBridgeABI, err := bridgegen.IBridgeMetaData.GetAbi()
	if err != nil {
		panic(err)
	}
	iBridgeABI = parsedIBridgeABI
	parsedIMessageProviderABI, err := bridgegen.IDelayedMessageProviderMetaData.GetAbi()
	if err != nil {
		panic(err)
	}
	iDelayedMessageProviderABI = parsedIMessageProviderABI
	inboxMessageDeliveredID = parsedIMessageProviderABI.Events["InboxMessageDelivered"].ID
	inboxMessageFromOriginID = parsedIMessageProviderABI.Events["InboxMessageDeliveredFromOrigin"].ID
	seqInboxABI, err = bridgegen.SequencerInboxMetaData.GetAbi()
	if err != nil {
		panic(err)
	}
	parsedIInboxABI, err := bridgegen.IInboxMetaData.GetAbi()
	if err != nil {
		panic(err)
	}
	iInboxABI = parsedIInboxABI
	batchDeliveredID = sequencerBridgeABI.Events["SequencerBatchDelivered"].ID
}
