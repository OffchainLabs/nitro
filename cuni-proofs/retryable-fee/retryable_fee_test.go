// Test: retryable submission fee vectors against the real implementation.
//
// The vectors in testdata/vectors.json were generated from a standalone
// transcription of the formula; this test runs the `submission_fee`
// vectors against the actual nitro implementation,
// retryables.RetryableSubmissionFee (arbos/retryables/retryable.go:394),
// and asserts exact equality in wei.
package cuniproofs

import (
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/offchainlabs/nitro/arbos/retryables"
)

type vector struct {
	Fn   string   `json:"fn"`
	Args []string `json:"args"`
	Want string   `json:"want"`
	Note string   `json:"note"`
}

func loadVectors(t *testing.T) []vector {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "vectors.json"))
	if err != nil {
		t.Fatalf("reading vectors: %v", err)
	}
	var vecs []vector
	if err := json.Unmarshal(raw, &vecs); err != nil {
		t.Fatalf("parsing vectors: %v", err)
	}
	return vecs
}

func TestSubmissionFeeVectors(t *testing.T) {
	n := 0
	for _, v := range loadVectors(t) {
		if v.Fn != "submission_fee" {
			continue
		}
		n++
		calldataLen, ok := new(big.Int).SetString(v.Args[0], 10)
		if !ok {
			t.Fatalf("vector %q: bad arg %q", v.Note, v.Args[0])
		}
		l1BaseFee, ok := new(big.Int).SetString(v.Args[1], 10)
		if !ok {
			t.Fatalf("vector %q: bad arg %q", v.Note, v.Args[1])
		}
		want, ok := new(big.Int).SetString(v.Want, 10)
		if !ok {
			t.Fatalf("vector %q: bad want %q", v.Note, v.Want)
		}
		if !calldataLen.IsInt64() {
			t.Fatalf("vector %q: calldata length %s out of int range", v.Note, v.Args[0])
		}
		got := retryables.RetryableSubmissionFee(int(calldataLen.Int64()), l1BaseFee)
		if got.Cmp(want) != 0 {
			t.Errorf("vector %q: RetryableSubmissionFee(%s, %s) = %s, want %s",
				v.Note, v.Args[0], v.Args[1], got.String(), want.String())
		}
	}
	if n == 0 {
		t.Fatal("no submission_fee vectors found")
	}
	t.Logf("%d submission_fee vectors match the implementation exactly", n)
}
