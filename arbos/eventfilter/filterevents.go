// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package eventfilter

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// BypassRule defines when to skip filtering entirely for an event.
type BypassRule struct {
	// Which topic to check (1-indexed, 0 is signature)
	TopicIndex int

	// Bypass when topic equals this address
	EqualsTo common.Address
}

// EventRule defines filtering behavior for a specific event type.
type EventRule struct {
	// Signature is the event signature hash (topics[0])
	Signature common.Hash

	// Topic indices containing addresses to filter (1-indexed)
	TopicAddresses []int

	// DataAddresses contains byte offsets in the event data where addresses are located
	DataAddresses []int

	// Bypass defines a rule to skip filtering for this event, nil if no bypass
	Bypass *BypassRule
}

type EventFilter struct {
	rules map[common.Hash]EventRule
}

func (b *BypassRule) UnmarshalJSON(data []byte) error {
	var raw struct {
		TopicIndex int    `json:"topicIndex"`
		EqualsTo   string `json:"equalsTo"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	b.TopicIndex = raw.TopicIndex
	b.EqualsTo = common.HexToAddress(raw.EqualsTo)
	return nil
}

func (e *EventRule) UnmarshalJSON(data []byte) error {
	var raw struct {
		Event          string      `json:"event"`
		TopicAddresses []int       `json:"topicAddresses"`
		DataAddresses  []int       `json:"dataAddresses,omitempty"`
		Bypass         *BypassRule `json:"bypass,omitempty"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	e.Signature = crypto.Keccak256Hash([]byte(raw.Event))
	e.TopicAddresses = raw.TopicAddresses
	e.DataAddresses = raw.DataAddresses
	e.Bypass = raw.Bypass
	return nil
}

func (r *EventRule) Validate() error {
	if r.Signature == (common.Hash{}) {
		return fmt.Errorf("empty event signature")
	}

	for i, idx := range r.TopicAddresses {
		if idx <= 0 || idx > 3 {
			return fmt.Errorf("topicAddresses[%d] out of range, got %d", i, idx)
		}
	}

	for i, offset := range r.DataAddresses {
		if offset < 0 || offset%32 != 0 {
			return fmt.Errorf("dataAddresses[%d]: offset must be non-negative and 32-byte aligned, got %d", i, offset)
		}
	}

	if r.Bypass != nil && (r.Bypass.TopicIndex <= 0 || r.Bypass.TopicIndex > 3) {
		return fmt.Errorf("bypass.topicIndex must be 1-3, got %d", r.Bypass.TopicIndex)
	}

	return nil
}

func NewEventFilter(rules []EventRule) (*EventFilter, error) {
	m := make(map[common.Hash]EventRule, len(rules))
	for i, rule := range rules {
		if err := rule.Validate(); err != nil {
			return nil, fmt.Errorf("rules[%d]: %w", i, err)
		}
		if _, exists := m[rule.Signature]; exists {
			return nil, fmt.Errorf("rules[%d]: duplicate signature %s", i, rule.Signature.Hex())
		}
		m[rule.Signature] = rule
	}
	return &EventFilter{rules: m}, nil
}

func NewEventFilterFromJSON(data []byte) (*EventFilter, error) {
	var config struct {
		Rules []EventRule `json:"rules"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	return NewEventFilter(config.Rules)
}

func NewEventFilterFromFile(path string) (*EventFilter, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}
	return NewEventFilterFromJSON(data)
}

// ExtractAddresses returns all addresses referenced by this event rule verbatim.
func (f *EventFilter) ExtractAddresses(topics []common.Hash, data []byte, _emitter common.Address, _sender common.Address) []common.Address {
	if len(topics) == 0 {
		return nil
	}

	rule, ok := f.rules[topics[0]]
	if !ok {
		return nil
	}

	if rule.Bypass != nil {
		idx := rule.Bypass.TopicIndex
		if idx > 0 && idx < len(topics) {
			if common.BytesToAddress(topics[idx][12:]) == rule.Bypass.EqualsTo {
				return nil
			}
		}
	}

	seen := make(map[common.Address]struct{})

	// Extract from topics
	for _, idx := range rule.TopicAddresses {
		if idx > 0 && idx < len(topics) {
			address := common.BytesToAddress(topics[idx][12:])
			seen[address] = struct{}{}
		}
	}

	// Extract from data
	for _, offset := range rule.DataAddresses {
		if offset >= 0 && offset+32 <= len(data) {
			address := common.BytesToAddress(data[offset+12 : offset+32])
			seen[address] = struct{}{}
		}
	}

	if len(seen) == 0 {
		return nil
	}

	out := make([]common.Address, 0, len(seen))
	for addr := range seen {
		out = append(out, addr)
	}
	return out
}
