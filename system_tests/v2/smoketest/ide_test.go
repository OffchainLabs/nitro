// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

// IDE entry points: one Test wrapper per scenario group.
package smoketest

import (
	"testing"

	"github.com/offchainlabs/nitro/system_tests/v2/systest"
)

func TestMain(m *testing.M) {
	systest.RunTestMain(m)
}

func TestTransfers(t *testing.T) { systest.RunGroup(t, transferTests) }

func TestDeployment(t *testing.T) { systest.RunGroup(t, deploymentTests) }

func TestP256Verify(t *testing.T) { systest.RunGroup(t, p256VerifyTests) }

func TestL1(t *testing.T) { systest.RunGroup(t, l1Tests) }

func TestMultiNode(t *testing.T) { systest.RunGroup(t, multiNodeTests) }
