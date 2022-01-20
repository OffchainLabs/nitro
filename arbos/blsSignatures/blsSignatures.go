//
// Copyright 2022, Offchain Labs, Inc. All rights reserved.
//

package blsSignatures

import (
	cryptorand "crypto/rand"
	"errors"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/bls12381"
	"math/big"
)

type blsStateType struct {
	g1            *bls12381.G1
	g2            *bls12381.G2
	pairingEngine *bls12381.Engine
}

var blsState blsStateType

func init() {
	blsState = blsStateType{
		g1:            bls12381.NewG1(),
		g2:            bls12381.NewG2(),
		pairingEngine: bls12381.NewPairingEngine(),
	}
}

type PublicKey struct {
	key           *bls12381.PointG2
	validityProof *bls12381.PointG1 // if this is nil, PK was made by aggregating verified keys
}

type PrivateKey *big.Int

type Signature *bls12381.PointG1

func GenerateKeys() (*PublicKey, PrivateKey, error) {
	seed, err := cryptorand.Int(cryptorand.Reader, blsState.g2.Q())
	if err != nil {
		return nil, nil, err
	}
	return InsecureDeterministicGenerateKeys(seed)
}

// Use for testing only.
func InsecureDeterministicGenerateKeys(seed *big.Int) (*PublicKey, PrivateKey, error) {
	privateKey := seed
	pubKey := &bls12381.PointG2{}
	blsState.g2.MulScalar(pubKey, blsState.g2.One(), privateKey)
	proof, err := KeyValidityProof(pubKey, privateKey)
	if err != nil {
		return nil, nil, err
	}
	publicKey, err := NewPublicKey(pubKey, proof)
	if err != nil {
		return nil, nil, err
	}
	return publicKey, privateKey, nil
}

// This key validity proof mechanism is sufficient to prevent rogue key attacks, if applied to all keys
//     that come from untrusted sources.
// See Theorem 1 in Ristenpart & Yilek, "The Power of Proofs-of-Possession: ..." from EUROCRYPT 2007.
func KeyValidityProof(pubKey *bls12381.PointG2, privateKey PrivateKey) (Signature, error) {
	return signMessage2(privateKey, blsState.g2.ToBytes(pubKey), true)
}

func NewPublicKey(pubKey *bls12381.PointG2, proof *bls12381.PointG1) (*PublicKey, error) {
	unverifiedPublicKey := &PublicKey{pubKey, proof}
	verified, err := verifySignature2(proof, blsState.g2.ToBytes(pubKey), unverifiedPublicKey, true)
	if err != nil {
		return nil, err
	}
	if !verified {
		return nil, errors.New("public key validation failed")
	}
	return unverifiedPublicKey, nil
}

func NewTrustedPublicKey(pubKey *bls12381.PointG2) *PublicKey {
	return &PublicKey{pubKey, nil}
}

func SignMessage(priv PrivateKey, message []byte) (Signature, error) {
	return signMessage2(priv, message, false)
}

func signMessage2(priv PrivateKey, message []byte, keyValidationMode bool) (Signature, error) {
	pointOnCurve, err := hashToG1Curve(message, keyValidationMode)
	if err != nil {
		return nil, err
	}
	result := &bls12381.PointG1{}
	blsState.g1.MulScalar(result, pointOnCurve, priv)
	return Signature(result), nil
}

func VerifySignature(sig Signature, message []byte, publicKey *PublicKey) (bool, error) {
	return verifySignature2(sig, message, publicKey, false)
}

func verifySignature2(sig Signature, message []byte, publicKey *PublicKey, keyValidationMode bool) (bool, error) {
	pointOnCurve, err := hashToG1Curve(message, keyValidationMode)
	if err != nil {
		return false, err
	}

	engine := blsState.pairingEngine
	engine.Reset()
	engine.AddPair(pointOnCurve, publicKey.key)
	leftSide := engine.Result()
	engine.AddPair(sig, blsState.g2.One())
	rightSide := engine.Result()
	return leftSide.Equal(rightSide), nil
}

func AggregatePublicKeys(pubKeys []*PublicKey) *PublicKey {
	ret := blsState.g2.Zero()
	for _, pk := range pubKeys {
		blsState.g2.Add(ret, ret, pk.key)
	}
	return NewTrustedPublicKey(ret)
}

func AggregateSignatures(sigs []Signature) Signature {
	ret := blsState.g1.Zero()
	for _, s := range sigs {
		blsState.g1.Add(ret, ret, s)
	}
	return ret
}

func VerifyAggregatedSignature(sig Signature, message []byte, pubKeys []*PublicKey) (bool, error) {
	return VerifySignature(sig, message, AggregatePublicKeys(pubKeys))
}

func VerifyAggregatedSignatureDifferentMessages(sig Signature, messages [][]byte, pubKeys []*PublicKey) (bool, error) {
	if len(messages) != len(pubKeys) {
		return false, errors.New("len(messages) does not match (len(pub keys) in verification")
	}
	engine := blsState.pairingEngine
	engine.Reset()
	for i, msg := range messages {
		pointOnCurve, err := hashToG1Curve(msg, false)
		if err != nil {
			return false, err
		}
		engine.AddPair(pointOnCurve, pubKeys[i].key)
	}
	leftSide := engine.Result()

	engine.Reset()
	engine.AddPair(sig, blsState.g2.One())
	rightSide := engine.Result()
	return leftSide.Equal(rightSide), nil
}

func hashToG1Curve(message []byte, keyValidationMode bool) (*bls12381.PointG1, error) {
	var empty [16]byte
	h := crypto.Keccak256(message)
	if keyValidationMode {
		h = append(h[16:], h[:16]...) // tweak the hash, so result won't collide with ordinary hash
	}
	return blsState.g1.MapToCurve(append(empty[:], h...))
}

func PublicKeyToBytes(pub PublicKey) []byte {
	if pub.validityProof == nil {
		return append([]byte{0}, blsState.g2.ToBytes(pub.key)...)
	} else {
		keyBytes := blsState.g2.ToBytes(pub.key)
		sigBytes := SignatureToBytes(pub.validityProof)
		if len(sigBytes) > 255 {
			panic("validity proof too large to serialize")
		}
		return append(append([]byte{byte(len(sigBytes))}, sigBytes...), keyBytes...)
	}
}

func PublicKeyFromBytes(in []byte, trustedSource bool) (*PublicKey, error) {
	sigLen := int(in[0])
	if sigLen == 0 {
		if !trustedSource {
			return nil, errors.New("tried to deserialize unvalidated public key from untrusted source")
		}
		key, err := blsState.g2.FromBytes(in[1:])
		if err != nil {
			return nil, err
		}
		return NewTrustedPublicKey(key), nil
	} else {
		if len(in) < 1+sigLen {
			return nil, errors.New("invalid serialized public key")
		}
		sigBytes := in[1 : 1+sigLen]
		sig, err := blsState.g1.FromBytes(sigBytes)
		if err != nil {
			return nil, err
		}
		keyBytes := in[1+sigLen:]
		key, err := blsState.g2.FromBytes(keyBytes)
		if err != nil {
			return nil, err
		}
		return NewPublicKey(key, sig)
	}
}

func PrivateKeyToBytes(priv PrivateKey) []byte {
	return ((*big.Int)(priv)).Bytes()
}

func PrivateKeyFromBytes(in []byte) (PrivateKey, error) {
	return new(big.Int).SetBytes(in), nil
}

func SignatureToBytes(sig Signature) []byte {
	return blsState.g1.ToBytes(sig)
}

func SignatureFromBytes(in []byte) (Signature, error) {
	return blsState.g1.FromBytes(in)
}
