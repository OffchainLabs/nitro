package dataposter

import (
	"bytes"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func Test_prepareTxTypeToPost(t *testing.T) {
	testhelpers.InitTestLog(t, 5)
	dataPoster := &DataPoster{}
	// TODO we may need to move the RLP encoding into prepareTxTypeToPost
	origL2MessageData := []byte("foobar")
	l2MessageData, err := rlp.EncodeToBytes(origL2MessageData)
	if err != nil {
		t.Fatal(err)
	}
	t.Run("EIP-4844 style txs contain valid blobs from L2 message data", func(t *testing.T) {
		dataPoster.isEip4844 = true
		preparedTx, networkBlobTx, err := dataPoster.prepareTxTypeToPost(
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

		if len(networkBlobTx.Blobs) != 1 {
			t.Fatalf("Expected a blob, got %d", len(networkBlobTx.Blobs))
		}
		versionedHashes := preparedTx.BlobHashes()
		if len(versionedHashes) != len(networkBlobTx.Blobs) {
			t.Fatalf("Num hashes %d != num blobs %d", len(versionedHashes), len(networkBlobTx.Blobs))
		}
		for i, k := range networkBlobTx.Commitments {
			computed := vm.KZGToVersionedHash(k)
			if versionedHashes[i] != computed {
				t.Fatalf(
					"hash at %d %#x != computed versioned hash %#x",
					i,
					versionedHashes[i],
					computed,
				)
			}
		}
		blob := networkBlobTx.Blobs[0]
		s := rlp.NewStream(bytes.NewReader(blob[1:]), 0) // TODO 1: is wrong, need to unpack blobs properly
		var decodedBytes []byte
		err = s.Decode(&decodedBytes)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(decodedBytes, origL2MessageData) {
			t.Fatalf("Wanted (%x) in blob, got (%x)", origL2MessageData, decodedBytes)
		}
		if strings.Contains(string(preparedTx.Data()), fmt.Sprintf("%x", l2MessageData)) {
			t.Fatal("Did not want tx calldata to contain L2 message data")
		}
	})
	/*
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
	*/
}
