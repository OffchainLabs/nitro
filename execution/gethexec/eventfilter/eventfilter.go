// Copyright 2026-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package eventfilter

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

const (
	eventSelectorSize = 4
	abiAddressOffset  = 12 // address is right-aligned in 32-byte ABI word
	logTopicCount     = 3  // topics[0] is event signature, topics[1..3] are indexed params
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
	Selector [eventSelectorSize]byte

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

var DefaultEventFilterConfig = EventFilterConfig{
	Path:  "",
	Rules: nil,
}

type EventFilter struct {
	rules map[[eventSelectorSize]byte]EventRule
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
		Bypass         *BypassRule `json:"bypass,omitempty"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// Parse selector
	selectorBytes := common.FromHex(raw.Selector)
	if len(selectorBytes) != eventSelectorSize {
		return fmt.Errorf("selector must be exactly %d bytes, got %d", eventSelectorSize, len(selectorBytes))
	}
	copy(e.Selector[:], selectorBytes)

	e.Event = raw.Event
	e.TopicAddresses = raw.TopicAddresses
	e.Bypass = raw.Bypass

	return nil
}

func EventRulesFromJSON(data []byte) ([]EventRule, error) {
	var rulesRaw struct {
		Rules []EventRule `json:"rules"`
	}
	if err := json.Unmarshal(data, &rulesRaw); err != nil {
		return nil, fmt.Errorf("parsing rules: %w", err)
	}
	return rulesRaw.Rules, nil
}

func EventRulesFromFile(path string) ([]EventRule, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}
	return EventRulesFromJSON(data)
}

func (e *EventRule) Validate() error {
	if e.Selector == ([eventSelectorSize]byte{}) {
		return fmt.Errorf("selector cannot be zero")
	}

	computed, canonical, err := CanonicalSelectorFromEvent(e.Event)
	if err != nil {
		return err
	}

	if e.Selector != computed {
		return fmt.Errorf(
			"event %q canonicalised to %q does not match selector 0x%s",
			e.Event,
			canonical,
			hex.EncodeToString(e.Selector[:]),
		)
	}

	for i, idx := range e.TopicAddresses {
		if idx <= 0 || idx > logTopicCount {
			return fmt.Errorf("topicAddresses[%d] out of range, got %d", i, idx)
		}
	}

	if e.Bypass != nil && (e.Bypass.TopicIndex <= 0 || e.Bypass.TopicIndex > logTopicCount) {
		return fmt.Errorf("bypass.topicIndex must be 1-%d, got %d", logTopicCount, e.Bypass.TopicIndex)
	}

	return nil
}

func (c *EventFilterConfig) Validate() error {
	if c.Rules == nil {
		return nil
	}

	seen := make(map[[eventSelectorSize]byte]struct{}, len(c.Rules))
	for i, rule := range c.Rules {
		if err := rule.Validate(); err != nil {
			return fmt.Errorf("validation: rule[%d] error: %w", i, err)
		}
		if _, ok := seen[rule.Selector]; ok {
			return fmt.Errorf("validation: duplicate rule selector 0x%s", hex.EncodeToString(rule.Selector[:]))
		}
		seen[rule.Selector] = struct{}{}
	}
	return nil
}

func NewEventFilter(rules []EventRule) (*EventFilter, error) {
	m := make(map[[eventSelectorSize]byte]EventRule, len(rules))
	for i, rule := range rules {
		if err := rule.Validate(); err != nil {
			return nil, fmt.Errorf("rule[%d] error: %w", i, err)
		}
		m[rule.Selector] = rule
	}
	return &EventFilter{rules: m}, nil
}

func NewEventFilterFromConfig(cfg EventFilterConfig) (*EventFilter, error) {
	var rules []EventRule

	if cfg.Path != "" {
		rulesFromFile, err := EventRulesFromFile(cfg.Path)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rulesFromFile...)
	}

	if len(cfg.Rules) != 0 {
		rules = append(rules, cfg.Rules...)
	}

	if len(rules) == 0 {
		return nil, nil
	}

	return NewEventFilter(rules)
}

func (f *EventFilter) HasRules() bool {
	return len(f.rules) > 0
}

// AddressesForFiltering returns all addresses referenced by this event rule verbatim.
func (f *EventFilter) AddressesForFiltering(topics []common.Hash, _data []byte, _emitter common.Address, _sender common.Address) []common.Address {
	if len(topics) == 0 {
		return []common.Address{}
	}

	var selector [eventSelectorSize]byte
	copy(selector[:], topics[0][:eventSelectorSize])

	rule, ok := f.rules[selector]
	if !ok {
		return []common.Address{}
	}

	if rule.Bypass != nil {
		idx := rule.Bypass.TopicIndex
		if idx > 0 && idx < len(topics) {
			if common.BytesToAddress(topics[idx][abiAddressOffset:]) == rule.Bypass.Equals {
				return []common.Address{}
			}
		}
	}

	seen := make(map[common.Address]struct{})

	// Extract from topics
	for _, idx := range rule.TopicAddresses {
		if idx > 0 && idx < len(topics) {
			address := common.BytesToAddress(topics[idx][abiAddressOffset:])
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

// CanonicalSelectorFromEvent parses an event signature, canonicalises its ABI types, and returns the selector and canonical form.
func CanonicalSelectorFromEvent(event string) (selector [eventSelectorSize]byte, canonical string, err error) {
	parsed, err := abi.ParseSelector(event)
	if err != nil {
		return selector, "", fmt.Errorf("invalid event %q: %w", event, err)
	}

	args := make([]string, len(parsed.Inputs))
	for i, in := range parsed.Inputs {
		var typ abi.Type
		typ, err = abi.NewType(in.Type, "", nil)
		if err != nil {
			return selector, "", fmt.Errorf("invalid type %q in event %q: %w", in.Type, event, err)
		}
		args[i] = typ.String()
	}

	canonical = fmt.Sprintf(
		"%s(%s)",
		parsed.Name,
		strings.Join(args, ","),
	)

	hash := crypto.Keccak256([]byte(canonical))
	copy(selector[:], hash[:eventSelectorSize])

	return selector, canonical, nil
}
