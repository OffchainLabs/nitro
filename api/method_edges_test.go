package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/OffchainLabs/bold/api"
	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	"github.com/OffchainLabs/bold/challenge-manager/challenge-tree/mock"
	"github.com/ethereum/go-ethereum/common"
	"gopkg.in/d4l3k/messagediff.v1"
)

func TestListEdges(t *testing.T) {
	s, d, _ := NewTestServer(t)

	d.Edges = []protocol.SpecEdge{
		&mock.Edge{
			ID:            mock.EdgeId(padHashString("foo")),
			EdgeType:      protocol.BlockChallengeEdge,
			StartHeight:   100,
			StartCommit:   mock.Commit(padHashString("foo_start_commit")),
			EndHeight:     150,
			EndCommit:     mock.Commit(padHashString("foo_end_commit")),
			OriginID:      mock.OriginId(padHashString("foo_origin_id")),
			ClaimID:       padHashString("foo_claim_id"),
			LowerChildID:  mock.EdgeId(padHashString("foo_lower_child_id")),
			UpperChildID:  mock.EdgeId(padHashString("foo_upper_child_id")),
			CreationBlock: 1,
		},
		&mock.Edge{
			ID:            mock.EdgeId(padHashString("bar")),
			EdgeType:      protocol.BigStepChallengeEdge,
			StartHeight:   110,
			StartCommit:   mock.Commit(padHashString("bar_start_commit")),
			EndHeight:     160,
			EndCommit:     mock.Commit(padHashString("bar_end_commit")),
			OriginID:      mock.OriginId(padHashString("bar_origin_id")),
			ClaimID:       padHashString("bar_claim_id"),
			LowerChildID:  mock.EdgeId(padHashString("bar_lower_child_id")),
			UpperChildID:  mock.EdgeId(padHashString("bar_upper_child_id")),
			CreationBlock: 2,
		},
		&mock.Edge{
			ID:            mock.EdgeId(padHashString("baz")),
			EdgeType:      protocol.SmallStepChallengeEdge,
			StartHeight:   111,
			StartCommit:   mock.Commit(padHashString("baz_start_commit")),
			EndHeight:     161,
			EndCommit:     mock.Commit(padHashString("baz_end_commit")),
			OriginID:      mock.OriginId(padHashString("baz_origin_id")),
			ClaimID:       padHashString("baz_claim_id"),
			LowerChildID:  mock.EdgeId(padHashString("baz_lower_child_id")),
			UpperChildID:  mock.EdgeId(padHashString("baz_upper_child_id")),
			CreationBlock: 5,
		},
	}

	req, err := http.NewRequest("GET", "/edges", nil)
	if err != nil {
		t.Fatal(err)
	}

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	// Serve the request with the http recorder.
	s.Router().ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Check the response body is what we expect.

	var resp []*api.Edge
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	respAsMockEdges := edgesToMockEdges(resp)

	if len(respAsMockEdges) != len(d.Edges) {
		t.Fatalf("Received different number of edges. Want %d, got %d", len(d.Edges), len(respAsMockEdges))
	}

	for i, re := range respAsMockEdges {
		if diff, ok := messagediff.PrettyDiff(re, d.Edges[i]); !ok {
			t.Errorf("Unexpected response at index %d. Diff: %s", i, diff)
		}
	}
}

func TestGetEdge(t *testing.T) {
	s, _, _ := NewTestServer(t)

	req, err := http.NewRequest("GET", "/edges/foo", nil)
	if err != nil {
		t.Fatal(err)
	}

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	// Serve the request with the http recorder.
	s.Router().ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusNotImplemented {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusNotImplemented)
	}
}

func padHashString(s string) string {
	return string(common.BytesToHash([]byte(s)).Bytes())
}

func TestPadHashString(t *testing.T) {
	s := padHashString("foobar")

	cID := protocol.ClaimId(common.BytesToHash([]byte(s)))

	ret := string(common.Hash(cID).Bytes())

	if s != ret {
		t.Log(len(s))
		t.Log(len(ret))
		t.Fatalf("Got %+v, wanted %+v", ret, s)
	}
}
