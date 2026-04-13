// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package programs

import (
	"testing"
)

func TestUpgradeToVersion(t *testing.T) {
	// Happy path: sequential 1 -> 2 -> 3 upgrade.
	p := &StylusParams{Version: 1}
	if err := p.UpgradeToVersion(2); err != nil {
		Fail(t, "expected version 1->2 upgrade to succeed:", err)
	}
	AssertEq(t, p.Version, uint16(2))
	AssertEq(t, p.MinInitGas, uint8(v2MinInitGas))

	if err := p.UpgradeToVersion(3); err != nil {
		Fail(t, "expected version 2->3 upgrade to succeed:", err)
	}
	AssertEq(t, p.Version, uint16(3))

	// Re-applying version 3 must fail.
	if err := p.UpgradeToVersion(3); err == nil {
		Fail(t, "expected version 3->3 re-application to fail")
	}

	// Skipping from 1 directly to 3 must fail; Version must be unchanged.
	p2 := &StylusParams{Version: 1}
	if err := p2.UpgradeToVersion(3); err == nil {
		Fail(t, "expected version 1->3 skip to fail")
	}
	AssertEq(t, p2.Version, uint16(1))

	// Re-applying version 2 must fail (p.Version != 1).
	p3 := &StylusParams{Version: 2}
	if err := p3.UpgradeToVersion(2); err == nil {
		Fail(t, "expected version 2->2 re-application to fail")
	}

	// Upgrading to an unsupported version must fail.
	p4 := &StylusParams{Version: 3}
	if err := p4.UpgradeToVersion(4); err == nil {
		Fail(t, "expected upgrade to unsupported version 4 to fail")
	}
}
