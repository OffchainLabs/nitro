// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package systest

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/trie"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/util/containers"
)

func TestWithValidationStampsBuilder(t *testing.T) {
	b := newBuilder()
	WithL1()(b)
	WithValidation()(b)
	b.validate()
	if !b.validation {
		t.Error("validation not set")
	}
	if got := b.freeze("").Topology; got != TopologyL1L2 {
		t.Errorf("topology = %v, want TopologyL1L2", got)
	}
	if w := b.weight(); w != weightMax {
		t.Errorf("weight = %d, want weightMax (%d)", w, weightMax)
	}
	if len(b.postHooks) != 1 {
		t.Errorf("postHooks = %d, want 1 (validateToHead)", len(b.postHooks))
	}
}

func TestValidateToHead(t *testing.T) {
	// Validation not requested → no-op (no validator needed).
	if err := validateToHead(&Env{Spec: Spec{Validate: false}}); err != nil {
		t.Errorf("Validate=false: want nil, got %v", err)
	}
	// Requested but the block validator isn't wired → hard error, never a silent pass.
	if err := validateToHead(&Env{Spec: Spec{Validate: true}}); err == nil {
		t.Error("Validate=true with nil validator: want error, got nil")
	}
}

func TestHasUsefulTx(t *testing.T) {
	mkBlock := func(txs ...*types.Transaction) *types.Block {
		return types.NewBlock(&types.Header{}, &types.Body{Transactions: txs}, nil, trie.NewStackTrie(nil))
	}
	internal := types.NewTx(&types.ArbitrumInternalTx{ChainId: big.NewInt(1)})
	useful := types.NewTx(&types.LegacyTx{})

	if hasUsefulTx(mkBlock()) {
		t.Fatal("empty block must not count as useful")
	}
	if hasUsefulTx(mkBlock(internal)) {
		t.Fatal("internal-tx-only block must not count as useful")
	}
	if !hasUsefulTx(mkBlock(internal, useful)) {
		t.Fatal("block with a non-internal tx must count as useful")
	}
}

func TestConfigureValidationRejectsConflictingScheme(t *testing.T) {
	execCfg := &gethexec.Config{}
	execCfg.Caching.StateScheme = "path"
	nodeCfg := &arbnode.Config{}
	if err := configureValidation(execCfg, nodeCfg, "ws://unused"); err == nil {
		t.Fatal("want error on a conflicting state-scheme override, got nil")
	}
}

func TestConfigureValidationRejectsEmptyServerConfigs(t *testing.T) {
	if err := configureValidation(&gethexec.Config{}, &arbnode.Config{}, "ws://unused"); err == nil {
		t.Fatal("want error when ValidationServerConfigs is empty, got nil")
	}
}

func TestConfigureValidationSetsUpNode(t *testing.T) {
	execCfg := &gethexec.Config{}
	nodeCfg := arbnode.ConfigDefaultL1Test()
	if err := configureValidation(execCfg, nodeCfg, "ws://valnode"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !execCfg.Caching.Archive {
		t.Error("Caching.Archive not set")
	}
	if execCfg.Caching.StateScheme != string(validationScheme) {
		t.Errorf("StateScheme = %q, want %q", execCfg.Caching.StateScheme, validationScheme)
	}
	if !nodeCfg.BlockValidator.Enable {
		t.Error("BlockValidator.Enable not set")
	}
	if url := nodeCfg.BlockValidator.ValidationServerConfigs[0].URL; url != "ws://valnode" {
		t.Errorf("ValidationServerConfigs[0].URL = %q, want ws://valnode", url)
	}
}

func TestValidationSchemePinConflictPanics(t *testing.T) {
	mustPanic(t, "validation requires hash", func() {
		b := newBuilder()
		WithL1()(b)
		WithStateScheme(StateSchemePath)(b)
		WithValidation()(b)
		b.validate()
	})
}

func TestValidationOptionPanics(t *testing.T) {
	mustPanic(t, "WithValidation applied twice", func() {
		b := newBuilder()
		WithValidation()(b)
		WithValidation()(b)
	})
	mustPanic(t, "requires a parent chain", func() {
		b := newBuilder()
		WithValidation()(b)
		b.validate()
	})
}

func TestValidationTopologyOrderIndependent(t *testing.T) {
	for name, opts := range map[string][]TestOption{
		"validation first": {WithValidation(), WithMultiNode()},
		"topology first":   {WithMultiNode(), WithValidation()},
	} {
		t.Run(name, func(t *testing.T) {
			b := newBuilder()
			for _, o := range opts {
				o(b)
			}
			b.validate()
			if got := b.freeze("").Topology; got != TopologyMultiNode {
				t.Fatalf("topology = %v, want TopologyMultiNode", got)
			}
		})
	}
}

func TestShouldSkipValidation(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(*builder)
		sp         scheduleParams
		wantReason string
	}{
		{
			name: "validation on resolved path cell",
			setup: func(b *builder) {
				b.validation = true
				b.stateScheme = containers.Some(StateSchemePath)
			},
			sp:         scheduleParams{},
			wantReason: "validation requires hash state scheme",
		},
		{
			name: "validation on env-default path",
			setup: func(b *builder) {
				b.validation = true
			},
			sp:         scheduleParams{DefaultStateScheme: containers.Some(StateSchemePath)},
			wantReason: "validation requires hash state scheme",
		},
		{
			name: "full-stack on forced path scheme",
			setup: func(b *builder) {
				WithFullStack()(b)
			},
			sp:         scheduleParams{StateScheme: containers.Some(StateSchemePath)},
			wantReason: "validation requires hash state scheme",
		},
		{
			name: "full-stack on env-default path",
			setup: func(b *builder) {
				WithFullStack()(b)
			},
			sp:         scheduleParams{DefaultStateScheme: containers.Some(StateSchemePath)},
			wantReason: "validation requires hash state scheme",
		},
	}
	for _, c := range tests {
		t.Run(c.name, func(t *testing.T) {
			b := newBuilder()
			c.setup(b)
			got := b.shouldSkip(c.sp)
			if got != c.wantReason {
				t.Fatalf("got %q, want %q", got, c.wantReason)
			}
		})
	}
}

func TestMatrixPathCellDeclinesValidation(t *testing.T) {
	b := newBuilder()
	b.name = "M"
	MatrixStateScheme(StateSchemeHash, StateSchemePath)(b)
	WithValidation()(b)
	dims := b.dims
	clone := b.clone()
	clone.dims = dims

	out := expandMatrix(clone, scheduleParams{})
	if len(out) != 2 {
		t.Fatalf("got %d cells, want 2", len(out))
	}
	for _, cell := range out {
		switch cell.Spec.StateScheme.Unwrap() {
		case StateSchemeHash:
			if cell.SkipReason != "" {
				t.Fatalf("hash cell skipped: %q", cell.SkipReason)
			}
			if !cell.Spec.Validate {
				t.Fatal("hash cell lost validation")
			}
		case StateSchemePath:
			if cell.SkipReason != "validation requires hash state scheme" {
				t.Fatalf("path cell: SkipReason = %q, want validation-requires-hash", cell.SkipReason)
			}
		default:
			t.Fatalf("unexpected cell scheme %q", cell.Spec.StateScheme.Unwrap())
		}
	}
}

func TestFullStackSchemePinConflictPanics(t *testing.T) {
	mustPanic(t, "validation requires hash", func() {
		b := newBuilder()
		WithStateScheme(StateSchemePath)(b)
		WithFullStack()(b)
		b.validate()
	})
}

func TestFullStackTopology(t *testing.T) {
	b := newBuilder()
	WithFullStack()(b)
	if b.topology != TopologyFullStack {
		t.Fatalf("topology = %v, want TopologyFullStack", b.topology)
	}
	if w := b.weight(); w != weightMax {
		t.Fatalf("weight = %d, want weightMax (%d)", w, weightMax)
	}

	// WithValidation must not downgrade a higher topology.
	WithValidation()(b)
	if b.topology != TopologyFullStack {
		t.Fatalf("WithValidation downgraded topology to %v", b.topology)
	}
}

func TestFullStackTopologyConflicts(t *testing.T) {
	cases := []struct {
		name string
		opts []TestOption
		want string
	}{
		{"WithFullStack twice", []TestOption{WithFullStack(), WithFullStack()}, "WithFullStack applied twice"},
		{"WithMultiNode then WithFullStack", []TestOption{WithMultiNode(), WithFullStack()}, "conflicts with another topology"},
		{"WithFullStack then WithL1", []TestOption{WithFullStack(), WithL1()}, "conflicts with another topology"},
		{"WithL1 then WithFullStack", []TestOption{WithL1(), WithFullStack()}, "conflicts with another topology"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			mustPanic(t, c.want, func() {
				b := newBuilder()
				for _, o := range c.opts {
					o(b)
				}
			})
		})
	}
}
