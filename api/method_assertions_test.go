package api_test

import (
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/OffchainLabs/bold/api"
	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	"github.com/ethereum/go-ethereum/common"
	"gopkg.in/d4l3k/messagediff.v1"
)

func TestListAssertions(t *testing.T) {
	s, _, provider := NewTestServer(t)

	provider.Hashes = []protocol.AssertionHash{
		{Hash: common.BigToHash(big.NewInt(0))},
		{Hash: common.BigToHash(big.NewInt(1))},
		{Hash: common.BigToHash(big.NewInt(2))},
	}
	provider.AssertionCreationInfos = []*protocol.AssertionCreatedInfo{
		{
			ConfirmPeriodBlocks: 55,
			RequiredStake:       big.NewInt(1e18),
			ParentAssertionHash: common.HexToHash("0xf00"),
			InboxMaxCount:       big.NewInt(120),
			AfterInboxBatchAcc:  common.HexToHash("0xa00"),
			AssertionHash:       common.HexToHash("0x12"),
			WasmModuleRoot:      common.HexToHash("0x11"),
			ChallengeManager:    common.HexToAddress("0x12"),
			TransactionHash:     common.HexToHash("0x13"),
			CreationBlock:       1,
			AfterState: (&protocol.ExecutionState{
				GlobalState: protocol.GoGlobalState{
					BlockHash:  common.HexToHash("0xb10"),
					SendRoot:   common.HexToHash("0xb20"),
					Batch:      1,
					PosInBatch: 0,
				},
			}).AsSolidityStruct(),
		},
		{
			ConfirmPeriodBlocks: 56,
			RequiredStake:       big.NewInt(1e18),
			ParentAssertionHash: common.HexToHash("0xf01"),
			InboxMaxCount:       big.NewInt(121),
			AfterInboxBatchAcc:  common.HexToHash("0xa01"),
			AssertionHash:       common.HexToHash("0x121"),
			WasmModuleRoot:      common.HexToHash("0x111"),
			ChallengeManager:    common.HexToAddress("0x121"),
			TransactionHash:     common.HexToHash("0x131"),
			CreationBlock:       2,
			AfterState: (&protocol.ExecutionState{
				GlobalState: protocol.GoGlobalState{
					BlockHash:  common.HexToHash("0xb11"),
					SendRoot:   common.HexToHash("0xb21"),
					Batch:      2,
					PosInBatch: 1,
				},
			}).AsSolidityStruct(),
		},
		{
			ConfirmPeriodBlocks: 57,
			RequiredStake:       big.NewInt(1e18),
			ParentAssertionHash: common.HexToHash("0xf002"),
			InboxMaxCount:       big.NewInt(122),
			AfterInboxBatchAcc:  common.HexToHash("0xa002"),
			AssertionHash:       common.HexToHash("0x122"),
			WasmModuleRoot:      common.HexToHash("0x112"),
			ChallengeManager:    common.HexToAddress("0x122"),
			TransactionHash:     common.HexToHash("0x132"),
			CreationBlock:       3,
			AfterState: (&protocol.ExecutionState{
				GlobalState: protocol.GoGlobalState{
					BlockHash:  common.HexToHash("0xb12"),
					SendRoot:   common.HexToHash("0xb22"),
					Batch:      5,
					PosInBatch: 2,
				},
			}).AsSolidityStruct(),
		},
	}

	inputsAsApiAssertions := make([]*api.Assertion, len(provider.AssertionCreationInfos))
	for i, aci := range provider.AssertionCreationInfos {
		inputsAsApiAssertions[i] = api.AssertionCreatedInfoToAssertion(aci)
	}

	req, err := http.NewRequest("GET", "/assertions", nil)
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
			status, http.StatusNotImplemented)
	}

	// Check the response body is what we expect.
	var resp []*api.Assertion
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	for i, a := range resp {
		if diff, ok := messagediff.PrettyDiff(a, inputsAsApiAssertions[i]); !ok {
			t.Errorf("Unexpected response at index %d. Diff: %s", i, diff)
		}
	}
}

func TestGetAssertion(t *testing.T) {
	s, _, _ := NewTestServer(t)

	req, err := http.NewRequest("GET", "/assertions/foo", nil)
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
