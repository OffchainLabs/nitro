// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package main

import (
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/arbitrum/multigas"
)

func ParseResourceKind(s string) (multigas.ResourceKind, error) {
	switch s {
	case "Computation":
		return multigas.ResourceKindComputation, nil
	case "HistoryGrowth":
		return multigas.ResourceKindHistoryGrowth, nil
	case "StorageAccess":
		return multigas.ResourceKindStorageAccess, nil
	case "StorageGrowth":
		return multigas.ResourceKindStorageGrowth, nil
	case "L1Calldata":
		return multigas.ResourceKindL1Calldata, nil
	case "L2Calldata":
		return multigas.ResourceKindL2Calldata, nil
	case "WasmComputation":
		return multigas.ResourceKindWasmComputation, nil
	default:
		return multigas.ResourceKindUnknown, fmt.Errorf("unknown resource kind: %s", s)
	}
}

type ResourceWeights map[uint8]uint64

func (rw *ResourceWeights) UnmarshalJSON(data []byte) error {
	weights := make(map[string]uint64)
	if err := json.Unmarshal(data, &weights); err != nil {
		return err
	}
	*rw = ResourceWeights{}
	for name, weight := range weights {
		kind, err := ParseResourceKind(name)
		if err != nil {
			return err
		}
		(*rw)[uint8(kind)] = weight
	}
	return nil
}

func (rw ResourceWeights) MarshalJSON() ([]byte, error) {
	weights := make(map[string]uint64)
	for id, weight := range rw {
		kind, err := multigas.CheckResourceKind(id)
		if err != nil {
			panic(fmt.Sprint("invalid resource id:", err))
		}
		name := kind.String()
		weights[name] = weight
	}
	return json.Marshal(weights)
}

type MultiGasConstraint struct {
	Target  uint64          `json:"target"`
	Window  uint32          `json:"window"`
	Backlog uint64          `json:"backlog"`
	Weights ResourceWeights `json:"weights"`
}

func (c MultiGasConstraint) Validate() error {
	if c.Target == 0 {
		return fmt.Errorf("target can't be 0")
	}
	if c.Window == 0 {
		return fmt.Errorf("adjustment window can't be 0")
	}
	if len(c.Weights) == 0 {
		return fmt.Errorf("constraint must have at least one weight")
	}
	return nil
}

const DefaultMultiGasConstraints string = `[
	{
		"target": 60000000,
		"window": 9,
		"backlog": 0,
		"weights": {
			"Computation": 1,
			"WasmComputation": 1,
			"HistoryGrowth": 1,
			"StorageAccess": 1,
			"StorageGrowth": 1,
			"L1Calldata": 1,
			"L2Calldata": 1
		}
	},

	{
		"target": 41000000,
		"window": 52,
		"backlog": 0,
		"weights": {
			"Computation": 1,
			"WasmComputation": 1,
			"HistoryGrowth": 1,
			"StorageAccess": 1,
			"StorageGrowth": 1,
			"L1Calldata": 1,
			"L2Calldata": 1
		}
	},

	{
		"target": 29000000,
		"window": 329,
		"backlog": 0,
		"weights": {
			"Computation": 1,
			"WasmComputation": 1,
			"HistoryGrowth": 1,
			"StorageAccess": 1,
			"StorageGrowth": 1,
			"L1Calldata": 1,
			"L2Calldata": 1
		}
	},

	{
		"target": 20000000,
		"window": 2105,
		"backlog": 0,
		"weights": {
			"Computation": 1,
			"WasmComputation": 1,
			"HistoryGrowth": 1,
			"StorageAccess": 1,
			"StorageGrowth": 1,
			"L1Calldata": 1,
			"L2Calldata": 1
		}
	},

	{
		"target": 14000000,
		"window": 13485,
		"backlog": 0,
		"weights": {
			"Computation": 1,
			"WasmComputation": 1,
			"HistoryGrowth": 1,
			"StorageAccess": 1,
			"StorageGrowth": 1,
			"L1Calldata": 1,
			"L2Calldata": 1
		}
	},

	{
		"target": 10000000,
		"window": 86400,
		"backlog": 0,
		"weights": {
			"Computation": 1,
			"WasmComputation": 1,
			"HistoryGrowth": 1,
			"StorageAccess": 1,
			"StorageGrowth": 1,
			"L1Calldata": 1,
			"L2Calldata": 1
		}
	}
]`
