// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package systest

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/util/containers"
)

func TestCloneConfigPreservesExecConfig(t *testing.T) {
	cfg := defaultExecConfig(t, containers.Some(StateSchemeHash))
	cloned := cloneConfig(cfg)
	if diff := diffStructs("gethexec.Config", *cfg, *cloned); diff != "" {
		t.Fatalf("cloneConfig dropped state — gob round-trip differs:\n%s\nEither Validate() must restore it, or the clone strategy must change.", diff)
	}
}

func TestCloneConfigPreservesNodeConfig(t *testing.T) {
	cfg := arbnode.ConfigDefaultL2Test()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("validate node config: %v", err)
	}
	cloned := cloneConfig(cfg)
	if diff := diffStructs("arbnode.Config", *cfg, *cloned); diff != "" {
		t.Fatalf("cloneConfig dropped state — gob round-trip differs:\n%s\nEither Validate() must restore it, or the clone strategy must change.", diff)
	}
}

// TestCloneConfigPreservesL1NodeConfig covers the config buildL1L2Node clones.
// The L1 default enables subtrees the L2 default leaves zero (BatchPoster,
// DelayedSequencer, SeqCoordinator, DA) — gob omits zero fields, so those
// non-zero paths are only exercised here, not by the L2 tripwire above.
func TestCloneConfigPreservesL1NodeConfig(t *testing.T) {
	cfg := arbnode.ConfigDefaultL1Test()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("validate L1 node config: %v", err)
	}
	cloned := cloneConfig(cfg)
	if diff := diffStructs("arbnode.Config(L1)", *cfg, *cloned); diff != "" {
		t.Fatalf("cloneConfig dropped state — gob round-trip differs:\n%s\nEither Validate() must restore it, or the clone strategy must change.", diff)
	}
}

// diffStructs reports field-level differences between two structs via
// reflection. Exported fields only — gob never encodes unexported fields, so
// the unexported class cloneConfig's CAUTION warns about can't be diffed here;
// those must be rebuilt by Validate().
func diffStructs(name string, a, b any) string {
	var diffs []string
	walkDiff(reflect.ValueOf(a), reflect.ValueOf(b), name, &diffs, 0)
	return strings.Join(diffs, "\n")
}

func walkDiff(a, b reflect.Value, path string, diffs *[]string, depth int) {
	if depth > 7 || len(*diffs) > 20 {
		return
	}
	if a.Kind() != b.Kind() {
		*diffs = append(*diffs, fmt.Sprintf("%s: kind %s vs %s", path, a.Kind(), b.Kind()))
		return
	}
	switch a.Kind() {
	case reflect.Struct:
		for i := range a.NumField() {
			f := a.Type().Field(i)
			if !f.IsExported() {
				continue
			}
			walkDiff(a.Field(i), b.Field(i), path+"."+f.Name, diffs, depth+1)
		}
	case reflect.Slice, reflect.Map:
		// Treat nil and empty as equivalent — gob normalizes empty slices to
		// nil, which is content-equivalent for our purposes.
		if a.Len() == 0 && b.Len() == 0 {
			return
		}
		if !reflect.DeepEqual(a.Interface(), b.Interface()) {
			*diffs = append(*diffs, fmt.Sprintf("%s: %v -> %v", path, a.Interface(), b.Interface()))
		}
	default:
		if !reflect.DeepEqual(a.Interface(), b.Interface()) {
			*diffs = append(*diffs, fmt.Sprintf("%s: %v -> %v", path, a.Interface(), b.Interface()))
		}
	}
}
