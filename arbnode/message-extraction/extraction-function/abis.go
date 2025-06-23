package extractionfunction

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
)

var inboxMessageDeliveredID common.Hash
var inboxMessageFromOriginID common.Hash
var iBridgeABI *abi.ABI
var iInboxABI *abi.ABI
var iDelayedMessageProviderABI *abi.ABI
var seqInboxABI *abi.ABI

func init() {
	var err error
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
	parsedIInboxABI, err := bridgegen.IInboxMetaData.GetAbi()
	if err != nil {
		panic(err)
	}
	iInboxABI = parsedIInboxABI
	seqInboxABI, err = bridgegen.SequencerInboxMetaData.GetAbi()
	if err != nil {
		panic(err)
	}
}
