package dataposter

import (
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func Test_prepareTxTypeToPost(t *testing.T) {
	dataPoster := &DataPoster[any]{}
	l2MessageData := []byte("foobar")
	t.Run("EIP-4844 style txs contain valid blobs from L2 message data", func(t *testing.T) {
		dataPoster.isEip4844 = true
		preparedTx, _, err := dataPoster.prepareTxTypeToPost(
			big.NewInt(1),
			big.NewInt(2),
			&DataToPost{
				SequencerInboxCalldata: []byte("fakecalldata"),
				L2MessageData:          l2MessageData,
			},
			3,
			common.HexToAddress("0xda1b6Bf463392eEF628a9d635ee20d1698Cf3Ca0"),
			4,
		)
		if err != nil {
			t.Fatal(err)
		}
		versionedHashes, kzgs, blobs, _ := preparedTx.BlobWrapData()
		if len(blobs) != 1 {
			t.Fatalf("Expected a blob, got %d", len(blobs))
		}
		if len(versionedHashes) != len(blobs) {
			t.Fatalf("Num hashes %d != num blobs %d", len(versionedHashes), len(blobs))
		}
		for i, k := range kzgs {
			computed := k.ComputeVersionedHash()
			if versionedHashes[i] != computed {
				t.Fatalf(
					"hash at %d %#x != computed versioned hash %#x",
					i,
					versionedHashes[i],
					computed,
				)
			}
		}
		blob := blobs[0]
		encodedText, err := blob.MarshalText()
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(encodedText), fmt.Sprintf("%x", l2MessageData)) {
			t.Fatalf("Wanted %s in blob, got %x", l2MessageData, encodedText)
		}
		if strings.Contains(string(preparedTx.Data()), fmt.Sprintf("%x", l2MessageData)) {
			t.Fatal("Did not want tx calldata to contain L2 message data")
		}
	})
	t.Run("Non-EIP-4844 style txs embed L2 message data within calldata", func(t *testing.T) {
		dataPoster.isEip4844 = true
		preparedTx, _, err := dataPoster.prepareTxTypeToPost(
			big.NewInt(1),
			big.NewInt(2),
			&DataToPost{
				SequencerInboxCalldata: l2MessageData,
			},
			3,
			common.HexToAddress("0xda1b6Bf463392eEF628a9d635ee20d1698Cf3Ca0"),
			4,
		)
		if err != nil {
			t.Fatal(err)
		}
		if string(preparedTx.Data()) != string(l2MessageData) {
			t.Fatalf("Wanted %s in calldata, got %s", l2MessageData, string(preparedTx.Data()))
		}
	})
}
