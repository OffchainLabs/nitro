package espressocrypto

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	espressoClient "github.com/EspressoSystems/espresso-network-go/client"
	lightclient "github.com/EspressoSystems/espresso-network-go/light-client"
	espressoTypes "github.com/EspressoSystems/espresso-network-go/types"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

type merkleProofTestData struct {
	Proof             json.RawMessage `json:"proof"`
	Header            json.RawMessage `json:"header"`
	BlockMerkleRoot   string          `json:"block_merkle_root"`
	HotShotCommitment []uint8         `json:"hotshot_commitment"`
}

type namespaceProofTestData struct {
	NsProof   json.RawMessage `json:"ns_proof"`
	VidCommit string          `json:"vid_commit"`
	VidCommon json.RawMessage `json:"vid_common"`
	Namespace uint64          `json:"namespace"`
	NsTable   []uint8         `json:"ns_table"`
	TxCommit  string          `json:"tx_commit"`
}

// go test ./espressocrypto -run ^TestGenerateMerkleProofTestData$
//
// Make sure the espresso network and L1 is running
// If you are using dev node, visit http://localhost:{port}/v0/api/dev-info to get the light client address
func IgnoreTestGenerateMerkleProofTestData(t *testing.T) {
	fmt.Println("Generating merkle proof test data...")
	ctx := context.Background()
	hotshotUrl := "http://localhost:41000"
	l1Url := "http://localhost:8545"
	lightClientAddr := "0x0f1f89aaf1c6fdb7ff9d361e4388f5f3997f12a8"

	tx := espressoTypes.Transaction{
		Namespace: 100,
		Payload:   []byte("test"),
	}

	hotshotClient := espressoClient.NewClient(hotshotUrl, hotshotUrl)
	txHash, err := hotshotClient.SubmitTransaction(ctx, tx)
	if err != nil {
		t.Fatalf("Failed to submit transaction: %v", err)
	}
	fmt.Println("Transaction submitted:", txHash)

	var txData espressoTypes.TransactionQueryData
	limit := 30
	for {
		txData, err = hotshotClient.FetchTransactionByHash(ctx, txHash)
		if err == nil {
			break
		}
		limit--
		if limit <= 0 {
			t.Fatalf("Failed to fetch transaction")
		}
		time.Sleep(1 * time.Second)
	}

	header, err := hotshotClient.FetchRawHeaderByHeight(ctx, txData.BlockHeight)
	if err != nil {
		t.Fatalf("Failed to fetch header: %v", err)
	}

	l1Client, err := ethclient.Dial(l1Url)
	if err != nil {
		t.Fatalf("Failed to dial L1 client: %v", err)
	}
	lightClientReader, err := lightclient.NewLightClientReader(common.HexToAddress(lightClientAddr), l1Client)
	if err != nil {
		t.Fatalf("Failed to create light client reader: %v", err)
	}

	var nextHeight uint64
	var commitment espressoTypes.Commitment
	limit = 30
	for {
		snapshot, err := lightClientReader.FetchMerkleRoot(txData.BlockHeight, &bind.CallOpts{})
		if err == nil && snapshot.Height > 0 {
			nextHeight = snapshot.Height
			commitment = snapshot.Root
			break
		}
		limit--
		if limit <= 0 {
			t.Fatalf("Failed to fetch merkle root")
		}
		time.Sleep(15 * time.Second)
	}

	fmt.Println("snapshot height:", nextHeight)
	fmt.Println("Fetching block merkle proof...")

	proof, err := hotshotClient.FetchBlockMerkleProof(ctx, nextHeight, txData.BlockHeight)
	if err != nil {
		t.Fatalf("Failed to fetch block merkle proof: %v", err)
	}

	nextHeader, err := hotshotClient.FetchHeaderByHeight(ctx, nextHeight)
	if err != nil {
		t.Fatalf("Failed to fetch header: %v", err)
	}

	testData := merkleProofTestData{
		Proof:             proof.Proof,
		Header:            header,
		BlockMerkleRoot:   nextHeader.Header.GetBlockMerkleTreeRoot().String(),
		HotShotCommitment: commitment[:],
	}

	filePath := "merkle_proof_test_data2.json"
	file, err := os.Create(filePath)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	defer file.Close()

	json.NewEncoder(file).Encode(testData)
}

/// For namespace proof testdata, it is recommended to generate
/// it in the `espresso-network` repo.
/// Please refer to the following commit for the example:
/// https://github.com/EspressoSystems/espresso-network/commit/796051340344ea7022d066f58edbc9cb657653aa
