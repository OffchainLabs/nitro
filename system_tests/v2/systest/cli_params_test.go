// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package systest

import (
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/params"
)

func TestCapacityForValidateSerializes(t *testing.T) {
	saved := *flagMaxWeight
	t.Cleanup(func() { *flagMaxWeight = saved })

	*flagMaxWeight = 0
	if got := capacityFor(scheduleParams{Validate: true}); got != int(weightMax) {
		t.Fatalf("validate + unset max-weight: got %d, want %d", got, int(weightMax))
	}
	*flagMaxWeight = 3
	if got := capacityFor(scheduleParams{Validate: true}); got != 3 {
		t.Fatalf("validate + max-weight=3: got %d, want 3", got)
	}
	*flagMaxWeight = 0
	if got, want := capacityFor(scheduleParams{Validate: false}), capacity(); got != want {
		t.Fatalf("non-validate: got %d, want capacity()=%d", got, want)
	}
}

func TestBuildCLIParamsValidation(t *testing.T) {
	cases := []struct {
		name                           string
		stateScheme, dbEngine          string
		mArbOS, mStates, mDBs, wantErr string
	}{
		{name: "valid empty"},
		{name: "bad state scheme", stateScheme: "bogus", wantErr: "invalid -systest.state-scheme"},
		{name: "bad db engine", dbEngine: "rocksdb", wantErr: "invalid -systest.db-engine"},
		{name: "bad matrix arbos", mArbOS: "30,abc", wantErr: "invalid -systest.matrix.arbos"},
		{name: "bad matrix state", mStates: "hash,bogus", wantErr: "invalid -systest.matrix.state-scheme"},
		{name: "bad matrix db", mDBs: "rocksdb", wantErr: "invalid -systest.matrix.db-engine"},
		{name: "dup matrix arbos", mArbOS: "30,30", wantErr: "duplicate -systest.matrix.arbos"},
		{name: "dup matrix state", mStates: "hash,hash", wantErr: "duplicate -systest.matrix.state-scheme"},
		{name: "dup matrix db", mDBs: "pebble,pebble", wantErr: "duplicate -systest.matrix.db-engine"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := buildCLIParams("", "default", 0, c.stateScheme, c.dbEngine, c.mArbOS, c.mStates, c.mDBs)
			if c.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), c.wantErr) {
				t.Fatalf("got %v, want error containing %q", err, c.wantErr)
			}
		})
	}
}

func TestBuildCLIParamsRejectsBadGlob(t *testing.T) {
	if _, err := buildCLIParams("Transfer[", "default", 0, "", "", "", "", ""); err == nil {
		t.Fatal("expected error for malformed -systest.tests glob")
	} else if !strings.Contains(err.Error(), "invalid -systest.tests pattern") {
		t.Fatalf("got %v, want invalid-pattern error", err)
	}
	if _, err := buildCLIParams("Transfer*,Deploy*", "default", 0, "", "", "", "", ""); err != nil {
		t.Fatalf("valid globs rejected: %v", err)
	}
}

func TestBuildCLIParamsMatrixParsing(t *testing.T) {
	p, err := buildCLIParams("", "default", 0, "", "", "30,40", "hash,path", "pebble")
	if err != nil {
		t.Fatal(err)
	}
	if len(p.MatrixArbOS) != 2 || p.MatrixArbOS[0] != params.ArbosVersion_30 || p.MatrixArbOS[1] != params.ArbosVersion_40 {
		t.Fatalf("MatrixArbOS = %v, want [30 40]", p.MatrixArbOS)
	}
	if len(p.MatrixStates) != 2 || len(p.MatrixDBs) != 1 {
		t.Fatalf("matrix states=%v dbs=%v, want 2 and 1", p.MatrixStates, p.MatrixDBs)
	}
}
