// Copyright 2021-2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbutil

type FinalityData struct {
	FinalizedMsgCount MessageIndex
	SafeMsgCount      MessageIndex
	ValidatedMsgCount *MessageIndex
}
