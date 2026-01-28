// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package eventfilter

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// BypassRule defines when to skip filtering entirely for an event.
type BypassRule struct {
	// Which topic to check (1-indexed, 0 is signature)
	TopicIndex int

	// Bypass when topic equals this address
	Equals common.Address
}

// EventRule defines filtering behavior for a specific event type.
type EventRule struct {
	// Event is the human-readable event signature, used for documentation and validation.
	Event string

	// Selector is the first 4 bytes of keccak256(Event).
	// It is used to identify which event rule applies to a given log.
	Selector [4]byte

	// Topic indices containing addresses to filter (1-indexed)
	TopicAddresses []int

	// Bypass defines a rule to skip filtering for this event, nil if no bypass
	Bypass *BypassRule
}

type EventFilterConfig struct {
	// Optional JSON file with event filter rules
	Path string `koanf:"path"`
	// Optional inline rules, appended after file rules (if any)
	Rules []EventRule `koanf:"rules"`
}

type EventFilter struct {
	rules map[[4]byte]EventRule
}

func (b *BypassRule) UnmarshalJSON(data []byte) error {
	var raw struct {
		TopicIndex int    `json:"topicIndex"`
		Equals     string `json:"equals"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	b.TopicIndex = raw.TopicIndex
	b.Equals = common.HexToAddress(raw.Equals)
	return nil
}

func (e *EventRule) UnmarshalJSON(data []byte) error {
	var raw struct {
		Event          string      `json:"event"`
		Selector       string      `json:"selector"`
		TopicAddresses []int       `json:"topicAddresses"`
		DataAddresses  []int       `json:"dataAddresses,omitempty"`
		Bypass         *BypassRule `json:"bypass,omitempty"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// Parse selector
	selectorBytes := common.FromHex(raw.Selector)
	if len(selectorBytes) != 4 {
		return fmt.Errorf("selector must be exactly 4 bytes, got %d", len(selectorBytes))
	}
	copy(e.Selector[:], selectorBytes)

	e.Event = raw.Event
	e.TopicAddresses = raw.TopicAddresses
	e.Bypass = raw.Bypass

	return nil
}

func (e *EventRule) Validate() error {
	if e.Selector == ([4]byte{}) {
		return fmt.Errorf("selector cannot be zero")
	}

	// Parse and canonicalise event signature
	parsed, err := abi.ParseSelector(e.Event)
	if err != nil {
		return fmt.Errorf("invalid event %q: %w", e.Event, err)
	}

	// Canonicalise argument list
	args := make([]string, len(parsed.Inputs))
	for i, in := range parsed.Inputs {
		typ, err := abi.NewType(in.Type, "", nil)
		if err != nil {
			return fmt.Errorf("invalid type %q in event %q: %w", in.Type, e.Event, err)
		}
		args[i] = typ.String()
	}

	canonical := fmt.Sprintf(
		"%s(%s)",
		parsed.Name,
		strings.Join(args, ","),
	)

	// Compute selector from canonical form
	hash := crypto.Keccak256([]byte(canonical))
	var computed [4]byte
	copy(computed[:], hash[:4])

	if e.Selector != computed {
		return fmt.Errorf(
			"event %q canonicalised to %q does not match selector 0x%x",
			e.Event,
			canonical,
			e.Selector,
		)
	}

	for i, idx := range e.TopicAddresses {
		if idx <= 0 || idx > 3 {
			return fmt.Errorf("topicAddresses[%d] out of range, got %d", i, idx)
		}
	}

	if e.Bypass != nil && (e.Bypass.TopicIndex <= 0 || e.Bypass.TopicIndex > 3) {
		return fmt.Errorf("bypass.topicIndex must be 1-3, got %d", e.Bypass.TopicIndex)
	}

	return nil
}

func (c *EventFilterConfig) Validate() error {
	if c == nil {
		return nil
	}

	seen := make(map[[4]byte]struct{}, len(c.Rules))
	for i, rule := range c.Rules {
		if err := rule.Validate(); err != nil {
			return fmt.Errorf("validation: rule[%d] error: %w", i, err)
		}
		if _, ok := seen[rule.Selector]; ok {
			return fmt.Errorf("validation: duplicate rule selector 0x%x", rule.Selector)
		}
		seen[rule.Selector] = struct{}{}
	}
	return nil
}

func NewEventFilter(rules []EventRule) (*EventFilter, error) {
	m := make(map[[4]byte]EventRule, len(rules))
	for i, rule := range rules {
		if err := rule.Validate(); err != nil {
			return nil, fmt.Errorf("rule[%d] error: %w", i, err)
		}
		m[rule.Selector] = rule
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

func NewEventFilterFromConfig(cfg *EventFilterConfig) (*EventFilter, error) {
	if cfg == nil {
		return nil, nil
	}

	var rules []EventRule

	if cfg.Path != "" {
		filter, err := NewEventFilterFromFile(cfg.Path)
		if err != nil {
			return nil, err
		}
		if filter != nil {
			for _, r := range filter.rules {
				rules = append(rules, r)
			}
		}
	}

	if len(cfg.Rules) != 0 {
		rules = append(rules, cfg.Rules...)
	}

	if len(rules) == 0 {
		return nil, nil
	}

	return NewEventFilter(rules)
}

// ExtractAddresses returns all addresses referenced by this event rule verbatim.
func (f *EventFilter) ExtractAddresses(topics []common.Hash, data []byte, _emitter common.Address, _sender common.Address) []common.Address {
	if len(topics) == 0 {
		return []common.Address{}
	}

	var selector [4]byte
	copy(selector[:], topics[0][:4])

	rule, ok := f.rules[selector]
	if !ok {
		return []common.Address{}
	}

	if rule.Bypass != nil {
		idx := rule.Bypass.TopicIndex
		if idx > 0 && idx < len(topics) {
			if common.BytesToAddress(topics[idx][12:]) == rule.Bypass.Equals {
				return []common.Address{}
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

	if len(seen) == 0 {
		return []common.Address{}
	}

	out := make([]common.Address, 0, len(seen))
	for addr := range seen {
		out = append(out, addr)
	}
	return out
}

// Helper function to compute selector from event signature, used in unit and system tests.
func Selector4(sig string) [4]byte {
	hash := crypto.Keccak256([]byte(sig))
	var out [4]byte
	copy(out[:], hash[:4])
	return out
}
