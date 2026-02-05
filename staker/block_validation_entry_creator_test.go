// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package staker

var _ BlockValidationEntryCreator = (*MELEnabledValidationEntryCreator)(nil)
var _ BlockValidationEntryCreator = (*PreMELValidationEntryCreator)(nil)
