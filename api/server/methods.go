package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/OffchainLabs/bold/api"
	"github.com/OffchainLabs/bold/api/db"
	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	"github.com/OffchainLabs/bold/state-commitments/history"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/gorilla/mux"
)

var contentType = "application/json"

// Healthz checks if the API server is ready to serve queries. Returns 200 if it is ready.
//
// method:
// - GET
// - /api/v1/db/healthz
func (s *Server) Healthz(w http.ResponseWriter, r *http.Request) {
	// TODO: Respond with a 503 if the client the BOLD validator is
	// connected to is syncing.
	w.WriteHeader(http.StatusOK)
}

// ListAssertions up to chain head
//
// method:
// - GET
// - /api/v1/assertions
//
// request query params:
//   - limit: the max number of items in the response
//   - offset: the offset index in the DB
//   - inbox_max_count: assertions that have a specified value for InboxMaxCount
//   - from_block_number: items that were created since a specific block number. Defaults to latest confirmed assertion
//   - to_block_number: caps the response to assertions up to and including a block number
//   - challenged: fetch only assertions that have been challenged
//   - force_update: refetch the updatable fields of each item in the response
//
// response:
// - []*JsonAssertion
func (s *Server) ListAssertions(w http.ResponseWriter, r *http.Request) {
	opts := make([]db.AssertionOption, 0)
	query := r.URL.Query()
	if val, ok := query["limit"]; ok && len(val) > 0 {
		if v, err := strconv.Atoi(val[0]); err == nil {
			opts = append(opts, db.WithAssertionLimit(v))
		}
	}
	if val, ok := query["offset"]; ok && len(val) > 0 {
		if v, err := strconv.Atoi(val[0]); err == nil {
			opts = append(opts, db.WithAssertionOffset(v))
		}
	}
	if val, ok := query["inbox_max_count"]; ok && len(val) > 0 {
		opts = append(opts, db.WithInboxMaxCount(strings.Join(val, "")))
	}
	if val, ok := query["from_block_number"]; ok && len(val) > 0 {
		if v, err := strconv.ParseUint(val[0], 10, 64); err == nil {
			opts = append(opts, db.FromAssertionCreationBlock(v))
		}
	}
	if val, ok := query["to_block_number"]; ok && len(val) > 0 {
		if v, err := strconv.ParseUint(val[0], 10, 64); err == nil {
			opts = append(opts, db.ToAssertionCreationBlock(v))
		}
	}
	if _, ok := query["challenged"]; ok {
		opts = append(opts, db.WithChallenge())
	}
	if _, ok := query["force_update"]; ok {
		opts = append(opts, db.WithAssertionForceUpdate())
	}
	assertions, err := s.backend.GetAssertions(r.Context(), opts...)
	if err != nil {
		http.Error(w, fmt.Sprintf("Could not get assertions from backend: %v", err), http.StatusInternalServerError)
		return
	}
	writeJSONResponse(w, assertions)
}

// CollectMachineHashes fetches all the collectMachineHashes calls that have been made
// along with their start and end times.
//
// method:
// - GET
// - /api/v1/state-provider/requests/collect-machine-hashes
//
// request query params:
//   - limit: the max number of items in the response
//   - offset: the offset index in the DB
//   - ongoing: fetch only collectMachineHashes calls that are ongoing or did not finish
//
// response:
// - []*JsonCollectMachineHashes
func (s *Server) CollectMachineHashes(w http.ResponseWriter, r *http.Request) {
	opts := make([]db.CollectMachineHashesOption, 0)
	query := r.URL.Query()
	if val, ok := query["limit"]; ok && len(val) > 0 {
		if v, err := strconv.Atoi(val[0]); err == nil {
			opts = append(opts, db.WithCollectMachineHashesLimit(v))
		}
	}
	if val, ok := query["offset"]; ok && len(val) > 0 {
		if v, err := strconv.Atoi(val[0]); err == nil {
			opts = append(opts, db.WithCollectMachineHashesOffset(v))
		}
	}
	if _, ok := query["ongoing"]; ok {
		opts = append(opts, db.WithCollectMachineHashesOngoing())
	}
	assertions, err := s.backend.GetCollectMachineHashes(r.Context(), opts...)
	if err != nil {
		http.Error(w, fmt.Sprintf("Could not get CollectMachineHashes from backend: %v", err), http.StatusInternalServerError)
		return
	}
	writeJSONResponse(w, assertions)
}

// AssertionByIdentifier since the latest confirmed assertion.
//
// method:
// - GET
// - /api/v1/assertions/<identifier>
//
// identifier options:
// - an assertion hash (0x-prefixed): gets the assertion by hash
// - "latest-confirmed": gets the latest confirmed assertion
//
// query params
//   - force_update: refetch the updatable fields of each item in the response
//
// response:
// - *JsonAssertion
func (s *Server) AssertionByIdentifier(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	identifier := vars["identifier"]

	var assertion *api.JsonAssertion
	opts := []db.AssertionOption{
		db.WithAssertionLimit(1),
	}
	query := r.URL.Query()
	if _, ok := query["force_update"]; ok {
		opts = append(opts, db.WithAssertionForceUpdate())
	}
	if identifier == "latest-confirmed" {
		a, err := s.backend.LatestConfirmedAssertion(r.Context())
		if err != nil {
			http.Error(w, fmt.Sprintf("Could not get latest confirmed assertion: %v", err), http.StatusInternalServerError)
			return
		}
		assertion = a
	} else {
		// Otherwise, get the assertion by hash.
		hash, err := hexutil.Decode(identifier)
		if err != nil {
			http.Error(w, fmt.Sprintf("Could not parse assertion hash: %v", err), http.StatusBadRequest)
			return
		}
		opts = append(opts, db.WithAssertionHash(protocol.AssertionHash{Hash: common.BytesToHash(hash)}))
		assertions, err := s.backend.GetAssertions(r.Context(), opts...)
		if err != nil {
			http.Error(w, fmt.Sprintf("Could not get assertions from backend: %v", err), http.StatusInternalServerError)
			return
		}
		if len(assertions) != 1 {
			http.Error(
				w,
				fmt.Sprintf("Got more than 1 matching assertion: got %d", len(assertions)),
				http.StatusInternalServerError,
			)
			return
		}
		assertion = assertions[0]
	}
	writeJSONResponse(w, assertion)
}

// AllChallengeEdges fetches all the edges corresponding to a challenged
// assertion with a specific hash. This assertion hash must be the "parent assertion"
// of two child assertions that originated a challenge.
//
// method:
// - GET
// - /api/v1/challenge/<assertion-hash>/edges
//
// identifier options:
// - 0x-prefixed assertion hash
//
// request query params:
// - limit: the max number of items in the response
// - offset: the offset index in the DB
// - status: filter edges that have status "confirmed", "confirmable", or "pending"
// - royal: boolean true or false to get royal edges. If not set, fetches all edges in the challenge.
// - root_edges: boolean true or false to filter out only root edges (those that have a claim_id)
// - rivaled: boolean true or false to get only rivaled edges
// - has_length_one_rival: boolean true or false to get only edges that have a length one rival
// - only_subchallenged_edges: boolean true or false to get only edges that have a subchallenge claiming them
// - from_block_number: items that were created since a specific block number.
// - to_block_number: caps the response to edges up to a block number
// - inherited_timer_geq: edges with an inherited timer greater than some N number of blocks
// - to_block_number: caps the response to edges up to a block number
// - origin_id: edges that have a 0x-prefixed origin id
// - mutual_id: edges that have a 0x-prefixed mutual id
// - claim_id: edges that have a 0x-prefixed claim id
// - start_height: edges with a start height
// - end_height: edges with an end height
// - start_commitment: edges with a start history commitment of format "height:hash", such as 32:0xdeadbeef
// - end_commitment: edges with an end history commitment of format "height:hash", such as 32:0xdeadbeef
// - challenge_level: edges in a specific challenge level. level 0 is the block challenge level
// - force_update: refetch the updatable fields of each item in the response
// response:
// - []*JsonEdge
func (s *Server) AllChallengeEdges(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	assertionHashStr := vars["assertion-hash"]
	hash, err := hexutil.Decode(assertionHashStr)
	if err != nil {
		http.Error(w, fmt.Sprintf("Could not parse assertion hash: %v", err), http.StatusBadRequest)
		return
	}
	assertionHash := protocol.AssertionHash{Hash: common.BytesToHash(hash)}
	opts := []db.EdgeOption{
		db.WithEdgeAssertionHash(assertionHash),
	}
	query := r.URL.Query()
	if val, ok := query["limit"]; ok && len(val) > 0 {
		if v, err2 := strconv.Atoi(val[0]); err2 == nil {
			opts = append(opts, db.WithLimit(v))
		}
	}
	if val, ok := query["offset"]; ok && len(val) > 0 {
		if v, err2 := strconv.Atoi(val[0]); err2 == nil {
			opts = append(opts, db.WithOffset(v))
		}
	}
	if val, ok := query["status"]; ok && len(val) > 0 {
		status, err2 := parseEdgeStatus(strings.Join(val, ""))
		if err2 != nil {
			http.Error(w, fmt.Sprintf("Could not parse status: %v", err2), http.StatusBadRequest)
			return
		}
		opts = append(opts, db.WithEdgeStatus(status))
	}
	if val, ok := query["royal"]; ok {
		v := strings.Join(val, "")
		if v == "false" {
			opts = append(opts, db.WithRoyal(false))
		} else if v == "true" {
			opts = append(opts, db.WithRoyal(true))
		}
	}
	if _, ok := query["has_length_one_rival"]; ok {
		opts = append(opts, db.WithLengthOneRival())
	}
	if val, ok := query["rivaled"]; ok {
		v := strings.Join(val, "")
		if v == "false" {
			opts = append(opts, db.WithRival(false))
		} else if v == "true" {
			opts = append(opts, db.WithRival(true))
		}
	}
	if _, ok := query["only_subchallenged_edges"]; ok {
		opts = append(opts, db.WithSubchallenge())
	}
	if _, ok := query["root_edges"]; ok {
		opts = append(opts, db.WithRootEdges())
	}
	if _, ok := query["force_update"]; ok {
		opts = append(opts, db.WithEdgeForceUpdate())
	}
	if val, ok := query["from_block_number"]; ok && len(val) > 0 {
		if v, err2 := strconv.ParseUint(val[0], 10, 64); err2 == nil {
			opts = append(opts, db.FromEdgeCreationBlock(v))
		}
	}
	if val, ok := query["to_block_number"]; ok && len(val) > 0 {
		if v, err2 := strconv.ParseUint(val[0], 10, 64); err2 == nil {
			opts = append(opts, db.ToEdgeCreationBlock(v))
		}
	}
	if val, ok := query["start_height"]; ok && len(val) > 0 {
		if v, err2 := strconv.ParseUint(val[0], 10, 64); err2 == nil {
			opts = append(opts, db.WithStartHeight(v))
		}
	}
	if val, ok := query["end_height"]; ok && len(val) > 0 {
		if v, err2 := strconv.ParseUint(val[0], 10, 64); err2 == nil {
			opts = append(opts, db.WithEndHeight(v))
		}
	}
	if val, ok := query["inherited_timer_geq"]; ok && len(val) > 0 {
		if v, err2 := strconv.ParseUint(val[0], 10, 64); err2 == nil {
			opts = append(opts, db.WithInheritedTimerGreaterOrEq(v))
		}
	}
	if val, ok := query["origin_id"]; ok && len(val) > 0 {
		hash, err2 := hexutil.Decode(strings.Join(val, ""))
		if err2 != nil {
			http.Error(w, fmt.Sprintf("Could not parse origin_id: %v", err2), http.StatusBadRequest)
			return
		}
		opts = append(opts, db.WithOriginId(protocol.OriginId(common.BytesToHash(hash))))
	}
	if val, ok := query["mutual_id"]; ok && len(val) > 0 {
		hash, err2 := hexutil.Decode(strings.Join(val, ""))
		if err2 != nil {
			http.Error(w, fmt.Sprintf("Could not parse mutual_id: %v", err2), http.StatusBadRequest)
			return
		}
		opts = append(opts, db.WithMutualId(protocol.MutualId(common.BytesToHash(hash))))
	}
	if val, ok := query["claim_id"]; ok && len(val) > 0 {
		hash, err2 := hexutil.Decode(strings.Join(val, ""))
		if err2 != nil {
			http.Error(w, fmt.Sprintf("Could not parse claim_id: %v", err2), http.StatusBadRequest)
			return
		}
		opts = append(opts, db.WithClaimId(protocol.ClaimId(common.BytesToHash(hash))))
	}
	if val, ok := query["start_commitment"]; ok && len(val) > 0 {
		commitStr := strings.Join(val, "")
		commitParts := strings.Split(commitStr, ":")
		if len(commitParts) != 2 {
			http.Error(w, "Wrong start history commitment format, wanted height:hash", http.StatusBadRequest)
			return
		}
		startHeight, err2 := strconv.ParseUint(commitParts[0], 10, 64)
		if err2 != nil {
			http.Error(w, fmt.Sprintf("Could not parse start commit height: %v", err2), http.StatusBadRequest)
			return
		}
		startHash, err2 := hexutil.Decode(commitParts[1])
		if err2 != nil {
			http.Error(w, fmt.Sprintf("Could not parse start commit hash: %v", err2), http.StatusBadRequest)
			return
		}
		opts = append(opts, db.WithStartHistoryCommitment(history.History{
			Height: startHeight,
			Merkle: common.BytesToHash(startHash),
		}))
	}
	if val, ok := query["end_commitment"]; ok && len(val) > 0 {
		commitStr := strings.Join(val, "")
		commitParts := strings.Split(commitStr, ":")
		if len(commitParts) != 2 {
			http.Error(w, "Wrong start history commitment format, wanted height:hash", http.StatusBadRequest)
			return
		}
		endHeight, err2 := strconv.ParseUint(commitParts[0], 10, 64)
		if err2 != nil {
			http.Error(w, fmt.Sprintf("Could not parse end commit height: %v", err2), http.StatusBadRequest)
			return
		}
		endHash, err2 := hexutil.Decode(commitParts[1])
		if err2 != nil {
			http.Error(w, fmt.Sprintf("Could not parse end commit hash: %v", err2), http.StatusBadRequest)
			return
		}
		opts = append(opts, db.WithEndHistoryCommitment(history.History{
			Height: endHeight,
			Merkle: common.BytesToHash(endHash),
		}))
	}
	if val, ok := query["challenge_level"]; ok && len(val) > 0 {
		if v, err2 := strconv.ParseUint(val[0], 10, 8); err2 == nil {
			opts = append(opts, db.WithChallengeLevel(uint8(v)))
		}
	}
	edges, err := s.backend.GetEdges(r.Context(), opts...)
	if err != nil {
		http.Error(w, fmt.Sprintf("Could not get edges from backend: %v", err), http.StatusInternalServerError)
		return
	}
	writeJSONResponse(w, edges)
}

func parseEdgeStatus(str string) (protocol.EdgeStatus, error) {
	s := strings.TrimSpace(strings.ToLower(str))
	if s == "pending" {
		return protocol.EdgePending, nil
	} else if s == "confirmed" {
		return protocol.EdgeConfirmed, nil
	}
	return protocol.EdgePending, errors.New("unknown edge status, expected pending or confirmed")
}

// EdgeByIdentifier fetches an edge by its specific id in a challenge.
//
// method:
// - GET
// - /api/v1/challenge/<assertion-hash>/edges/id/<edge-id>
//
// identifier options:
// - 0x-prefixed assertion hash
// - 0x-prefixed edge id
//
// query params:
// - force_update: refetch the updatable fields of each item in the response
//
// response:
// - *JsonEdge
func (s *Server) EdgeByIdentifier(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	assertionHashStr := vars["assertion-hash"]
	edgeIdStr := vars["edge-id"]
	hash, err := hexutil.Decode(assertionHashStr)
	if err != nil {
		http.Error(w, fmt.Sprintf("Could not parse assertion hash: %v", err), http.StatusBadRequest)
		return
	}
	id, err := hexutil.Decode(edgeIdStr)
	if err != nil {
		http.Error(w, fmt.Sprintf("Could not parse edge id: %v", err), http.StatusBadRequest)
		return
	}
	assertionHash := protocol.AssertionHash{Hash: common.BytesToHash(hash)}
	edgeId := protocol.EdgeId{Hash: common.BytesToHash(id)}
	opts := []db.EdgeOption{
		db.WithLimit(1),
		db.WithEdgeAssertionHash(assertionHash),
		db.WithId(edgeId),
	}
	query := r.URL.Query()
	if _, ok := query["force_update"]; ok {
		opts = append(opts, db.WithEdgeForceUpdate())
	}
	edges, err := s.backend.GetEdges(
		r.Context(),
		opts...,
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("Could not get edges from backend: %v", err), http.StatusInternalServerError)
		return
	}
	if len(edges) != 1 {
		http.Error(w, fmt.Sprintf("Got more edges than expected: %d", len(edges)), http.StatusInternalServerError)
		return
	}
	writeJSONResponse(w, edges[0])
}

// RoyalTrackedChallengeEdges dumps the locally-tracked, royal edges kept in-memory by the BOLD software.
//
// method:
// - GET
// - /api/v1/tracked/royal-edges
func (s *Server) RoyalTrackedChallengeEdges(w http.ResponseWriter, r *http.Request) {
	resp, err := s.backend.GetTrackedRoyalEdges(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("Could not get tracked royal edges: %v", err), http.StatusInternalServerError)
		return
	}
	writeJSONResponse(w, resp)
}

// EdgeByHistoryCommitment fetches an edge by its specific history commitment in a challenge.
//
// method:
// - GET
// - /api/v1/challenge/<assertion-hash>/edges/history/<history-commitment>
//
// identifier options:
//   - 0x-prefixed assertion hash
//   - history commitment with the format startheight:starthash:endheight:endhash, such as
//     0:0xdeadbeef:32:0xdeadbeef
//
// query params:
// - force_update: refetch the updatable fields of each item in the response
//
// response:
// - *JsonEdge
func (s *Server) EdgeByHistoryCommitment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	assertionHashStr := vars["assertion-hash"]
	hash, err := hexutil.Decode(assertionHashStr)
	if err != nil {
		http.Error(w, fmt.Sprintf("Could not parse assertion hash: %v", err), http.StatusBadRequest)
		return
	}
	assertionHash := protocol.AssertionHash{Hash: common.BytesToHash(hash)}
	historyCommitment := vars["history-commitment"]
	parts := strings.Split(historyCommitment, ":")
	if len(parts) != 4 {
		http.Error(w, "Wrong history commitment format, wanted startheight:starthash:endheight:endhash", http.StatusBadRequest)
		return
	}
	startHeight, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		http.Error(w, fmt.Sprintf("Could not parse start height: %v", err), http.StatusBadRequest)
		return
	}
	startHash, err := hexutil.Decode(parts[1])
	if err != nil {
		http.Error(w, fmt.Sprintf("Could not parse start hash: %v", err), http.StatusBadRequest)
		return
	}
	endHeight, err := strconv.ParseUint(parts[2], 10, 64)
	if err != nil {
		http.Error(w, fmt.Sprintf("Could not parse end height: %v", err), http.StatusBadRequest)
		return
	}
	endHash, err := hexutil.Decode(parts[3])
	if err != nil {
		http.Error(w, fmt.Sprintf("Could not parse end hash: %v", err), http.StatusBadRequest)
		return
	}
	opts := []db.EdgeOption{
		db.WithEdgeAssertionHash(assertionHash),
		db.WithStartHistoryCommitment(history.History{
			Height: startHeight,
			Merkle: common.BytesToHash(startHash),
		}),
		db.WithEndHistoryCommitment(history.History{
			Height: endHeight,
			Merkle: common.BytesToHash(endHash),
		}),
		db.WithLimit(1),
	}
	query := r.URL.Query()
	if _, ok := query["force_update"]; ok {
		opts = append(opts, db.WithEdgeForceUpdate())
	}
	edges, err := s.backend.GetEdges(
		r.Context(),
		opts...,
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("Could not get edges from backend: %v", err), http.StatusInternalServerError)
		return
	}
	if len(edges) != 1 {
		http.Error(w, fmt.Sprintf("Got more edges than expected: %d", len(edges)), http.StatusInternalServerError)
		return
	}
	writeJSONResponse(w, edges[0])
}

// MiniStakes fetches all the mini-stakes present in a single challenged assertion.
//
// method:
// - GET
// - /api/v1/challenge/<assertion-hash>/ministakes
//
// identifier options:
//   - 0x-prefixed assertion hash
//
// request query params:
// - limit: the max number of items in the response
// - offset: the offset index in the DB
// - force_update: refetch the updatable fields of each item in the response
// - challenge_level: items in a specific challenge level. level 0 is the block challenge level
// response:
// - []*MiniStake
func (s *Server) MiniStakes(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	assertionHashStr := vars["assertion-hash"]
	hash, err := hexutil.Decode(assertionHashStr)
	if err != nil {
		http.Error(w, fmt.Sprintf("Could not parse assertion hash: %v", err), http.StatusBadRequest)
		return
	}
	assertionHash := protocol.AssertionHash{Hash: common.BytesToHash(hash)}
	query := r.URL.Query()
	opts := make([]db.EdgeOption, 0)
	if val, ok := query["limit"]; ok && len(val) > 0 {
		if v, err2 := strconv.Atoi(val[0]); err2 == nil {
			opts = append(opts, db.WithLimit(v))
		}
	}
	if val, ok := query["offset"]; ok && len(val) > 0 {
		if v, err2 := strconv.Atoi(val[0]); err2 == nil {
			opts = append(opts, db.WithOffset(v))
		}
	}
	if val, ok := query["challenge_level"]; ok && len(val) > 0 {
		if v, err2 := strconv.ParseUint(val[0], 10, 8); err2 == nil {
			opts = append(opts, db.WithChallengeLevel(uint8(v)))
		}
	}
	if _, ok := query["force_update"]; ok {
		opts = append(opts, db.WithEdgeForceUpdate())
	}
	miniStakes, err := s.backend.GetMiniStakes(r.Context(), assertionHash, opts...)
	if err != nil {
		http.Error(w, fmt.Sprintf("Could not get ministakes from backend: %v", err), http.StatusInternalServerError)
		return
	}
	writeJSONResponse(w, miniStakes)
}

func writeJSONResponse(w http.ResponseWriter, data any) {
	body, err := json.Marshal(data)
	if err != nil {
		http.Error(w, fmt.Sprintf("Could not write response: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(body)
	if err != nil {
		log.Error("could not write response body", "err", err, "status", http.StatusInternalServerError)
		return
	}
}
