// Copyright 2024, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package blobs

import (
	"bytes"
	"math/big"
	"math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/params"
)

const bytesEncodedPerBlob = 254 * 4096 / 8

var blsModulus, _ = new(big.Int).SetString("52435875175126190479447740508185965837690552500527637822603658699938581184513", 10)

func TestBlobEncoding(t *testing.T) {
	r := rand.New(rand.NewSource(1))
outer:
	for i := 0; i < 40; i++ {
		data := make([]byte, r.Int()%bytesEncodedPerBlob*3)
		_, err := r.Read(data)
		if err != nil {
			t.Fatalf("failed to generate random bytes: %v", err)
		}
		enc, err := EncodeBlobs(data)
		if err != nil {
			t.Errorf("failed to encode blobs for length %v: %v", len(data), err)
			continue
		}
		for _, b := range enc {
			for fieldElement := 0; fieldElement < params.BlobTxFieldElementsPerBlob; fieldElement++ {
				bigInt := new(big.Int).SetBytes(b[fieldElement*32 : (fieldElement+1)*32])
				if bigInt.Cmp(blsModulus) >= 0 {
					t.Errorf("for length %v blob %v has field element %v value %v >= modulus %v", len(data), b, fieldElement, bigInt, blsModulus)
					continue outer
				}
			}
		}
		dec, err := DecodeBlobs(enc)
		if err != nil {
			t.Errorf("failed to decode blobs for length %v: %v", len(data), err)
			continue
		}
		if !bytes.Equal(data, dec) {
			t.Errorf("got different decoding for length %v", len(data))
			continue
		}
	}
}
