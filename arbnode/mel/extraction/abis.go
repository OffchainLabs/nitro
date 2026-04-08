// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package melextraction

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
)

var BatchDeliveredID common.Hash
var InboxMessageDeliveredID common.Hash
var InboxMessageFromOriginID common.Hash
var MELConfigEventID common.Hash
var SeqInboxABI *abi.ABI
var IBridgeABI *abi.ABI
var iInboxABI *abi.ABI
var iDelayedMessageProviderABI *abi.ABI

func init() {
	var err error
	sequencerBridgeABI, err := bridgegen.SequencerInboxMetaData.GetAbi()
	if err != nil {
		panic(err)
	}
	BatchDeliveredID = sequencerBridgeABI.Events["SequencerBatchDelivered"].ID
	parsedIBridgeABI, err := bridgegen.IBridgeMetaData.GetAbi()
	if err != nil {
		panic(err)
	}
	IBridgeABI = parsedIBridgeABI
	parsedIMessageProviderABI, err := bridgegen.IDelayedMessageProviderMetaData.GetAbi()
	if err != nil {
		panic(err)
	}
	iDelayedMessageProviderABI = parsedIMessageProviderABI
	InboxMessageDeliveredID = parsedIMessageProviderABI.Events["InboxMessageDelivered"].ID
	InboxMessageFromOriginID = parsedIMessageProviderABI.Events["InboxMessageDeliveredFromOrigin"].ID
	SeqInboxABI, err = bridgegen.SequencerInboxMetaData.GetAbi()
	if err != nil {
		panic(err)
	}
	parsedIInboxABI, err := bridgegen.IInboxMetaData.GetAbi()
	if err != nil {
		panic(err)
	}
	iInboxABI = parsedIInboxABI

	// MELConfigEvent(address inbox, address sequencerInbox, uint256 melVersionActivationBlock)
	MELConfigEventID = crypto.Keccak256Hash([]byte("MELConfigEvent(address,address,uint256)"))
}
