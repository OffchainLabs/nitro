package arbnode

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/offchainlabs/nitro/util/signature"
)

func TestRecoverAddressFromSigner(t *testing.T) {
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	address, err := recoverAddressFromSigner(signature.DataSignerFromPrivateKey(privateKey))
	if err != nil {
		t.Fatal(err)
	}
	if address != crypto.PubkeyToAddress(privateKey.PublicKey) {
		t.Fatalf("expected address %v, got %v", crypto.PubkeyToAddress(privateKey.PublicKey), address)
	}
}

func TestBinarySearchForBlockNumber(t *testing.T) {
	target := uint64(64)
	count := 0
	ctx := context.Background()
	start := uint64(0)
	end := uint64(100)
	f := func(ctx context.Context, blockNumber uint64) (int, error) {
		count++
		if blockNumber < target {
			return -1, nil
		} else if blockNumber > target {
			return 1, nil
		}
		return 0, nil
	}
	result, err := binarySearchForBlockNumber(ctx, start, end, f)
	if err != nil {
		t.Fatal(err)
	}
	if result != target {
		t.Fatalf("expected result %d, got %d", target, result)
	}
	if count > 7 {
		t.Fatalf("expected count less than %d, got %d", 7, count)
	}

	targetRangeStart := uint64(60)
	targetRangeEnd := uint64(70)
	count = 0
	f = func(ctx context.Context, blockNumber uint64) (int, error) {
		count++
		if blockNumber < targetRangeStart {
			return -1, nil
		} else if blockNumber > targetRangeEnd {
			return 1, nil
		}
		return 0, nil
	}
	result, err = binarySearchForBlockNumber(ctx, start, end, f)
	if err != nil {
		t.Fatal(err)
	}
	if result != targetRangeStart {
		t.Fatalf("expected result %d, got %d", targetRangeStart, result)
	}
	if count > 7 {
		t.Fatalf("expected count less than %d, got %d", 7, count)
	}
}
