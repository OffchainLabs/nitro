// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package blsSignatures

import (
	"encoding/base64"
	"errors"
	bls12381 "github.com/consensys/gnark-crypto/ecc/bls12-381"
	"github.com/consensys/gnark-crypto/ecc/bls12-381/fp"
	"github.com/consensys/gnark-crypto/ecc/bls12-381/fr"
	"github.com/ethereum/go-ethereum/crypto"
	"math/big"
)

type PublicKey struct {
	key           *bls12381.G2Affine
	validityProof *bls12381.G1Affine // if this is nil, key came from a trusted source
}

type PrivateKey *fr.Element

type Signature *bls12381.G1Affine

func GeneratePrivKeyString() (string, error) {
	fr := new(fr.Element)
	privKey, err := fr.SetRandom()
	if err != nil {
		return "", err
	}
	privKeyBytes := PrivateKeyToBytes(privKey)
	encodedPrivKey := make([]byte, base64.StdEncoding.EncodedLen(len(privKeyBytes)))
	base64.StdEncoding.Encode(encodedPrivKey, privKeyBytes)
	return string(encodedPrivKey), nil
}

func GenerateKeys() (PublicKey, PrivateKey, error) {
	fr := new(fr.Element)
	privateKey, err := fr.SetRandom()
	if err != nil {
		return PublicKey{}, nil, err
	}
	publicKey, err := PublicKeyFromPrivateKey(privateKey)
	return publicKey, privateKey, err
}

func PublicKeyFromPrivateKey(privateKey PrivateKey) (PublicKey, error) {
	pubKey := new(bls12381.G2Affine)
	g2 := new(bls12381.G2Affine)
	g2.X.SetOne()
	g2.Y.SetOne()
	fr := new(fr.Element)
	fr.Set(privateKey)
	pubKey.ScalarMultiplication(g2, fr.BigInt(new(big.Int)))
	proof, err := KeyValidityProof(pubKey, privateKey)
	if err != nil {
		return PublicKey{}, err
	}
	publicKey, err := NewPublicKey(pubKey, proof)
	if err != nil {
		return PublicKey{}, err
	}
	return publicKey, nil
}

// KeyValidityProof is the key validity proof mechanism is sufficient to prevent rogue key attacks, if applied to all keys
// that come from untrusted sources. We use the private key to sign the public key, but in the
// signature algorithm we use a tweaked version of the hash-to-curve function so that the result cannot be
// re-used as an ordinary signature.
//
// For a proof that this is sufficient, see Theorem 1 in
// Ristenpart & Yilek, "The Power of Proofs-of-Possession: ..." from EUROCRYPT 2007.
func KeyValidityProof(pubKey *bls12381.G2Affine, privateKey PrivateKey) (Signature, error) {
	message := pubKey.Bytes()
	return signMessage2(privateKey, message[:], true)
}

func NewPublicKey(pubKey *bls12381.G2Affine, validityProof *bls12381.G1Affine) (PublicKey, error) {
	message := pubKey.Bytes()
	unverifiedPublicKey := PublicKey{pubKey, validityProof}
	verified, err := verifySignature2(validityProof, message[:], unverifiedPublicKey, true)
	if err != nil {
		return PublicKey{}, err
	}
	if !verified {
		return PublicKey{}, errors.New("public key validation failed")
	}
	return unverifiedPublicKey, nil
}

func NewTrustedPublicKey(pubKey *bls12381.G2Affine) PublicKey {
	return PublicKey{pubKey, nil}
}

func (pubKey PublicKey) ToTrusted() PublicKey {
	if pubKey.validityProof == nil {
		return pubKey
	}
	return NewTrustedPublicKey(pubKey.key)
}

func SignMessage(priv PrivateKey, message []byte) (Signature, error) {
	return signMessage2(priv, message, false)
}

func signMessage2(priv PrivateKey, message []byte, keyValidationMode bool) (Signature, error) {
	pointOnCurve, err := hashToG1Curve(message, keyValidationMode)
	if err != nil {
		return nil, err
	}
	result := new(bls12381.G1Affine)
	fr := new(fr.Element)
	fr.Set(priv)
	result.ScalarMultiplication(pointOnCurve, fr.BigInt(new(big.Int)))
	return result, nil
}

func VerifySignature(sig Signature, message []byte, publicKey PublicKey) (bool, error) {
	return verifySignature2(sig, message, publicKey, false)
}

func verifySignature2(sig Signature, message []byte, publicKey PublicKey, keyValidationMode bool) (bool, error) {
	pointOnCurve, err := hashToG1Curve(message, keyValidationMode)
	if err != nil {
		return false, err
	}

	leftSide, err := bls12381.Pair([]bls12381.G1Affine{*pointOnCurve}, []bls12381.G2Affine{*publicKey.key})
	if err != nil {
		return false, err
	}
	g2 := new(bls12381.G2Affine)
	g2.X.SetOne()
	g2.Y.SetOne()
	rightSide, err := bls12381.Pair([]bls12381.G1Affine{*sig}, []bls12381.G2Affine{*g2})
	if err != nil {
		return false, err
	}
	return leftSide.Equal(&rightSide), nil
}

func AggregatePublicKeys(pubKeys []PublicKey) PublicKey {
	g2 := new(bls12381.G2Affine)
	g2.X.SetZero()
	g2.Y.SetZero()
	for _, pk := range pubKeys {
		g2.Add(g2, pk.key)
	}
	return NewTrustedPublicKey(g2)
}

func AggregateSignatures(sigs []Signature) Signature {
	g1 := new(bls12381.G1Affine)
	g1.X.SetZero()
	g1.Y.SetZero()
	for _, s := range sigs {
		g1.Add(g1, s)
	}
	return g1
}

func VerifyAggregatedSignatureSameMessage(sig Signature, message []byte, pubKeys []PublicKey) (bool, error) {
	return VerifySignature(sig, message, AggregatePublicKeys(pubKeys))
}

func VerifyAggregatedSignatureDifferentMessages(sig Signature, messages [][]byte, pubKeys []PublicKey) (bool, error) {

	if len(messages) != len(pubKeys) {
		return false, errors.New("len(messages) does not match (len(pub keys) in verification")
	}
	var p []bls12381.G1Affine
	var q []bls12381.G2Affine
	for i, msg := range messages {
		pointOnCurve, err := hashToG1Curve(msg, false)
		if err != nil {
			return false, err
		}
		p = append(p, *pointOnCurve)
		q = append(q, *pubKeys[i].key)
	}
	leftSide, err := bls12381.Pair(p, q)
	if err != nil {
		return false, err
	}
	g2 := new(bls12381.G2Affine)
	g2.X.SetOne()
	g2.Y.SetOne()
	rightSide, err := bls12381.Pair([]bls12381.G1Affine{*sig}, []bls12381.G2Affine{*g2})
	if err != nil {
		return false, err
	}
	return leftSide.Equal(&rightSide), nil
}

// This hashes a message to a [32]byte, then maps the result to the G1 curve using
// the Simplified Shallue-van de Woestijne-Ulas Method, described in Section 6.6.2 of
// https://tools.ietf.org/html/draft-irtf-cfrg-hash-to-curve-06
//
// If keyValidationMode is true, this uses a tweaked version of the padding,
// so that the result will not collide with a result generated in an ordinary signature.
func hashToG1Curve(message []byte, keyValidationMode bool) (*bls12381.G1Affine, error) {
	var padding [16]byte
	h := crypto.Keccak256(message)
	if keyValidationMode {
		// modify padding, for domain separation
		padding[0] = 1
	}
	fp := new(fp.Element)
	fp.Unmarshal(append(padding[:], h...))
	res := bls12381.MapToG1(*fp)
	return &res, nil
}

func PublicKeyToBytes(pub PublicKey) []byte {
	keyBytes := pub.key.Bytes()
	if pub.validityProof == nil {
		return append([]byte{0}, keyBytes[:]...)
	}
	sigBytes := SignatureToBytes(pub.validityProof)
	if len(sigBytes) > 255 {
		panic("validity proof too large to serialize")
	}
	return append(append([]byte{byte(len(sigBytes))}, sigBytes...), keyBytes[:]...)
}

func PublicKeyFromBytes(in []byte, trustedSource bool) (PublicKey, error) {
	if len(in) == 0 {
		return PublicKey{}, errors.New("tried to deserialize empty public key")
	}
	key := new(bls12381.G2Affine)
	proofLen := int(in[0])
	if proofLen == 0 {
		if !trustedSource {
			return PublicKey{}, errors.New("tried to deserialize unvalidated public key from untrusted source")
		}
		err := key.Unmarshal(in[1:])
		if err != nil {
			return PublicKey{}, err
		}
		return NewTrustedPublicKey(key), nil
	} else {
		if len(in) < 1+proofLen {
			return PublicKey{}, errors.New("invalid serialized public key")
		}
		validityProof := new(bls12381.G1Affine)
		proofBytes := in[1 : 1+proofLen]
		err := validityProof.Unmarshal(proofBytes)
		if err != nil {
			return PublicKey{}, err
		}
		keyBytes := in[1+proofLen:]
		err = key.Unmarshal(keyBytes)
		if err != nil {
			return PublicKey{}, err
		}
		if trustedSource {
			// Skip verification of the validity proof
			return PublicKey{key, validityProof}, nil
		}
		return NewPublicKey(key, validityProof)
	}
}

func PrivateKeyToBytes(priv PrivateKey) []byte {
	bytes := new(fr.Element).Set(priv).Bytes()
	return bytes[:]
}

func PrivateKeyFromBytes(in []byte) (PrivateKey, error) {
	return new(fr.Element).SetBytes(in), nil
}

func SignatureToBytes(sig Signature) []byte {
	g1 := new(bls12381.G1Affine)
	g1.Set(sig)
	bytes := g1.Bytes()
	return bytes[:]
}

func SignatureFromBytes(in []byte) (Signature, error) {
	g1 := new(bls12381.G1Affine)
	err := g1.Unmarshal(in)
	if err != nil {
		return nil, err
	}
	return g1, nil
}
