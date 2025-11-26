// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package precompiles

// Defines the relative contribution of a specific resource type within a constraint.
type WeightedResource = struct {
	Resource uint8
	Weight   uint64
}

// Describes a single pricing constraint that applies to one or more weighted resources.
type MultiGasConstraint = struct {
	Resources            []WeightedResource
	AdjustmentWindowSecs uint32
	TargetPerSec         uint64
	Backlog              uint64
}
