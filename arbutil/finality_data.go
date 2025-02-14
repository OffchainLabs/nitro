// Copyright 2021-2025, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbutil

type FinalityData struct {
	FinalizedMsgCount MessageIndex
	SafeMsgCount      MessageIndex
	ValidatedMsgCount *MessageIndex
}
