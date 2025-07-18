package arbtest

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"net/http"
	"os/exec"
	"testing"
	"time"

	"github.com/prysmaticlabs/go-ssz"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

var timeBoostHealth = "/healthz"
var timeBoostSubmit = "/submit-regular"
var timeboostUrls = []string{
	"http://localhost:8800/v0", "http://localhost:8801/v0",
}

func runDecentralizedTimeboost() func() {
	shutdown := func() {
		log.Warn("shutdown timeboost docker")
		p := exec.Command("docker", "compose", "-f", "docker-compose.timeboost.yml", "down", "--volumes")
		p.Dir = workingDir
		var stderr bytes.Buffer
		p.Stderr = &stderr
		if err := p.Run(); err != nil {
			log.Error("failed to run 'docker compose down`", "err", err, "str", stderr.String())
			panic(err)
		}
		time.Sleep(5 * time.Second)
	}
	shutdown()

	cmd := exec.Command("docker", "compose", "-f", "docker-compose.timeboost.yml", "up", "-d")
	cmd.Dir = workingDir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		log.Error("failed to run 'docker compose up`", "err", err, "str", stderr.String())
		panic(err)
	}

	return shutdown
}

func waitForTimeboostNodes(ctx context.Context) error {
	for _, timeboostUrl := range timeboostUrls {
		if err := waitForWith(ctx, 1*time.Minute, 1*time.Second, func() bool {
			resp, err := http.Get(timeboostUrl + timeBoostHealth)
			if err != nil {
				log.Warn("retry to check the timeboost health", "err", err)
				return false
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				log.Warn("retry to check the timeboost health", "code", resp.StatusCode)
				return false
			}
			return true
		}); err != nil {
			return err
		}
	}

	return nil
}

type Bundle struct {
	Chain     int      `json:"chain"`
	Epoch     uint64   `json:"epoch"`
	Data      string   `json:"data"`
	Encrypted bool     `json:"encrypted"`
	Hash      [32]byte `json:"hash"`
}

func NewBundle(chain int, epoch uint64, data []byte, hash common.Hash) Bundle {
	return Bundle{
		Chain:     chain,
		Epoch:     epoch,
		Data:      "0x" + hex.EncodeToString(data),
		Encrypted: false,
		Hash:      hash,
	}
}

func createAndSendBundleToTimeboost(t *testing.T, builder *NodeBuilder, users []string) []*types.Transaction {
	var expectedTxs []*types.Transaction
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	// Various test cases at a given index in the user loop
	twoTxnsInBundleIdx := 8            // Send two txns in a bundle
	sendTxnToOneTimeboostNodeIdx := 10 // Only send the bundle to one timeboost node
	for i, userName := range users {
		tx := builder.L2Info.PrepareTx("Owner", userName, builder.L2Info.TransferGas, big.NewInt(2), nil)
		expectedTxs = append(expectedTxs, tx)
		txBytes, err := tx.MarshalBinary()
		Require(t, err)
		var encoded []byte
		if i == twoTxnsInBundleIdx {
			// Send 2 transactions in a bundle
			tx = builder.L2Info.PrepareTx("Owner", userName, builder.L2Info.TransferGas, big.NewInt(2), nil)
			expectedTxs = append(expectedTxs, tx)
			txBytes2, err := tx.MarshalBinary()
			Require(t, err)
			encoded, err = ssz.Marshal([][]byte{txBytes, txBytes2})
			Require(t, err)
		} else {
			encoded, err = ssz.Marshal([][]byte{txBytes})
			Require(t, err)
		}

		current := time.Now().Unix()
		if current < 0 {
			t.Fatalf("Invalid time %d", current)
		}
		epoch := uint64(current)
		bundle := NewBundle(0, epoch, encoded, tx.Hash())
		jsonData, err := json.MarshalIndent(bundle, "", "  ")
		Require(t, err)

		// Send to both nodes
		for _, timeboostUrl := range timeboostUrls {
			url := timeboostUrl + timeBoostSubmit
			req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
			Require(t, err)

			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Accept", "application/json")
			_, err = client.Do(req)
			Require(t, err)
			if i == sendTxnToOneTimeboostNodeIdx {
				// Only send to one node
				// This should still be fine and include the transaction
				continue
			}
		}
		time.Sleep(1 * time.Second)
	}
	return expectedTxs
}

func TestEspressoTimeboostSequencerE2E(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	valNodeCleanup := createValidationNode(ctx, t, true)
	defer valNodeCleanup()
	// In future, we also need to create a version of
	// delayed sequencer for timeboost
	builder, cleanup := createL1AndL2NodeForTimeboost(ctx, t, true)
	defer cleanup()

	err := waitForL1Node(ctx)
	Require(t, err)

	shutdown := runDecentralizedTimeboost()
	defer shutdown()

	err = waitForTimeboostNodes(ctx)
	Require(t, err)

	var users []string
	const numUsers = 15

	blockNumberBefore, err := builder.L2.Client.BlockNumber(ctx)
	Require(t, err)

	for num := 0; num < numUsers; num++ {
		userName := fmt.Sprintf("My_User_%d", num)
		builder.L2Info.GenerateAccount(userName)
		users = append(users, userName)
	}

	expectedTxs := createAndSendBundleToTimeboost(t, builder, users)
	if len(expectedTxs) != numUsers+1 {
		t.Fatalf("expected transactions should be num users + 1. num users %d, expected len %d", numUsers, len(expectedTxs))
	}
	// Wait for sometime for the blocks to be produced
	time.Sleep(time.Second * 10)

	blockNumberAfter, err := builder.L2.Client.BlockNumber(ctx)
	Require(t, err)

	// msgCntAfter should be greater than msgCntBefore
	if blockNumberAfter-blockNumberBefore <= 0 {
		t.Fatalf("expected difference between blockNumberAfter and blockNumberBefore to be greater than 0, got: %d", blockNumberAfter-blockNumberBefore)
	}

	// Insanity check
	if blockNumberAfter > math.MaxInt64 {
		t.Fatalf("expected blockNumberAfter to be less than max int64, got: %d", blockNumberAfter)
	}

	var transactions []*types.Transaction
	for i := blockNumberBefore + 1; i <= blockNumberAfter; i++ {
		if i > math.MaxInt64 {
			t.Fatalf("expected blockNumberAfter to be less than max int64, got: %d", blockNumberAfter)
		}
		block, err := builder.L2.Client.BlockByNumber(ctx, big.NewInt(int64(i)))
		Require(t, err)
		blockTransactions := block.Transactions()
		transactionsWithoutStartBlock := blockTransactions[1:]
		transactions = append(transactions, transactionsWithoutStartBlock...)
	}

	if len(transactions) != len(expectedTxs) {
		t.Fatalf("expected transactions and block transactions to match. got %d expected txns, got %d block transactions", len(expectedTxs), len(transactions))
	}

	for i, tx := range transactions {
		expected := expectedTxs[i]
		if tx.Hash() != expected.Hash() {
			t.Fatalf("txHash doesn't match, got %s, want %s.", tx.Hash().Hex(), expected.Hash().Hex())
		}
	}
}
