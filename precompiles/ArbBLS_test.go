//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package precompiles

import (
	"github.com/offchainlabs/arbstate/arbos/blsSignatures"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestArbBLS_12381(t *testing.T) {
	evm := newMockEVMForTesting()
	abls := ArbBLS{}
	addr1 := common.BytesToAddress([]byte{24})
	addr2 := common.BytesToAddress([]byte{42})
	context1 := testContext(addr1, evm)
	context2 := testContext(addr2, evm)

	_, err := abls.GetBLS12381(context1, evm, addr2)
	if err == nil {
		Fail(t)
	}

	pubKey2, privKey, err := blsSignatures.GenerateKeys()
	Require(t, err)
	err = abls.RegisterBLS12381(context2, evm, blsSignatures.PublicKeyToBytes(pubKey2))
	Require(t, err)

	recoveredPubKeyBytes2, err := abls.GetBLS12381(context1, evm, addr2)
	Require(t, err)
	recoveredPubKey2, err := blsSignatures.PublicKeyFromBytes(recoveredPubKeyBytes2, false)
	Require(t, err)

	msg := []byte{3, 1, 4, 1, 5, 9, 2, 6}
	sig, err := blsSignatures.SignMessage(privKey, msg)
	Require(t, err)
	success, err := blsSignatures.VerifySignature(sig, msg, recoveredPubKey2)
	Require(t, err)
	if !success {
		t.Fail()
	}
}
