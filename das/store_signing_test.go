// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package das

import (
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/offchainlabs/nitro/util/signature"
)

func TestStoreSigning(t *testing.T) {
	privateKey, err := crypto.GenerateKey()
	Require(t, err)

	addr := crypto.PubkeyToAddress(privateKey.PublicKey)

	weirdMessage := []byte("The quick brown fox jumped over the lazy dog.")
	timeout := uint64(time.Now().Unix())

	signer := signature.DataSignerFromPrivateKey(privateKey)
	sig, err := applyDasSigner(signer, weirdMessage, timeout)
	Require(t, err)

	recoveredAddr, err := DasRecoverSigner(weirdMessage, timeout, sig)
	Require(t, err)

	if recoveredAddr != addr {
		t.Fatal()
	}
}
