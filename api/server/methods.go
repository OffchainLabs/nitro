package server

// Healthz checks if the API server is ready to serve queries. Returns 200
// if it is ready, otherwise, other statuses. If the DB is currently reindexing
// data, it may return a 503.
//
// method:
// - GET
// - /api/v1/db/healthz
func (s *Server) Healthz() {

}

// ForceDBUpdate causes the DB scraper to reindex and fetch data
// from onchain and the challenge manager to update tables in the database.
//
// method:
// - POST
// - /api/v1/db/update
func (s *Server) ForceDBUpdate() {

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
//
// response:
// - []*JsonAssertion
func (s *Server) ListAssertions() {

}

// AssertionByIdentifier since the latest confirmed assertion.
//
// method:
// - GET
// - /api/v1/assertion/<identifier>
//
// identifier options:
// - an assertion hash (0x-prefixed): gets the assertion by hash
// - "latest-confirmed": gets the latest confirmed assertion
//
// response:
// - *JsonAssertion
func (s *Server) AssertionByIdentifier() {

}

// ChallengeByAssertionHash fetches information about a challenge on a specific assertion hash
// method:
// - GET
// - /api/v1/challenge/<assertion-hash>
//
// identifier options:
// - 0x-prefixed assertion hash
//
// response:
// - *JsonChallenge
func (s *Server) ChallengeByAssertionHash() {}

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
// - honest: boolean true or false to filter out honest vs. evil edges. If not set, fetches all edges in the challenge.
// - root_edges: boolean true or false to filter out only root edges (those that have a claim id)
// - from_block_number: items that were created since a specific block number. Defaults to challenge creation block
// - to_block_number: caps the response to edges up to and including a block number
// response:
// - origin_id: edges that have a 0x-prefixed origin id
// - mutual_id: edges that have a 0x-prefixed mutual id
// - claim_id: edges that have a 0x-prefixed claim id
// - start_commitment: edges with a start history commitment of format "height:hash", such as 32:0xdeadbeef
// - end_commitment: edges with an end history commitment of format "height:hash", such as 32:0xdeadbeef
// - challenge_level: edges in a specific challenge level. level 0 is the block challenge level
// - to_block_number: caps the response to edges up to and including a block number
// response:
// - []*JsonEdge
func (s *Server) AllChallengeEdges() {

}

// EdgeByIdentifier fetches an edge by its specific id in a challenge.
//
// method:
// - GET
// - /api/v1/challenge/<assertion-hash>/edges/<edge-id>
//
// identifier options:
// - 0x-prefixed assertion hash
// - 0x-prefixed edge id
//
// response:
// - *JsonEdge
func (s *Server) EdgeByIdentifier() {

}

// EdgeByHistoryCommitment fetches an edge by its specific history commitment in a challenge.
//
// method:
// - GET
// - /api/v1/challenge/<assertion-hash>/edges/<history-commitment>
//
// identifier options:
//   - 0x-prefixed assertion hash
//   - history commitment with the format startheight:starthash:endheight:endhash, such as
//     0:0xdeadbeef:32:0xdeadbeef
//
// response:
// - *JsonEdge
func (s *Server) EdgeByHistoryCommitment() {

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
// - challenge_level: items in a specific challenge level. level 0 is the block challenge level
// response:
// - []*MiniStake
func (s *Server) MiniStakes() {

}
