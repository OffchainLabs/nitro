// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package blsSignatures

import (
	cryptorand "crypto/rand"
	"encoding/base64"
	"errors"
	"github.com/ethereum/go-ethereum/crypto"
	bls12381 "github.com/kilic/bls12-381"
)

type PublicKey struct {
	key           *bls12381.PointG2
	validityProof *bls12381.PointG1 // if this is nil, key came from a trusted source
}

type PrivateKey *bls12381.Fr

type Signature *bls12381.PointG1

func GeneratePrivKeyString() (string, error) {
	fr := bls12381.NewFr()
	privKey, err := fr.Rand(cryptorand.Reader)
	if err != nil {
		return "", err
	}
	privKeyBytes := PrivateKeyToBytes(privKey)
	encodedPrivKey := make([]byte, base64.StdEncoding.EncodedLen(len(privKeyBytes)))
	base64.StdEncoding.Encode(encodedPrivKey, privKeyBytes)
	return string(encodedPrivKey), nil
}

func GenerateKeys() (PublicKey, PrivateKey, error) {
	fr := bls12381.NewFr()
	privateKey, err := fr.Rand(cryptorand.Reader)
	if err != nil {
		return PublicKey{}, nil, err
	}
	publicKey, err := PublicKeyFromPrivateKey(privateKey)
	return publicKey, privateKey, err
}

func PublicKeyFromPrivateKey(privateKey PrivateKey) (PublicKey, error) {
	pubKey := &bls12381.PointG2{}
	g2 := bls12381.NewG2()
	g2.MulScalar(pubKey, g2.One(), privateKey)
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
func KeyValidityProof(pubKey *bls12381.PointG2, privateKey PrivateKey) (Signature, error) {
	g2 := bls12381.NewG2()
	return signMessage2(privateKey, g2.ToBytes(pubKey), true)
}

func NewPublicKey(pubKey *bls12381.PointG2, validityProof *bls12381.PointG1) (PublicKey, error) {
	g2 := bls12381.NewG2()
	unverifiedPublicKey := PublicKey{pubKey, validityProof}
	verified, err := verifySignature2(validityProof, g2.ToBytes(pubKey), unverifiedPublicKey, true)
	if err != nil {
		return PublicKey{}, err
	}
	if !verified {
		return PublicKey{}, errors.New("public key validation failed")
	}
	return unverifiedPublicKey, nil
}

func NewTrustedPublicKey(pubKey *bls12381.PointG2) PublicKey {
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
	g1 := bls12381.NewG1()
	result := &bls12381.PointG1{}
	g1.MulScalar(result, pointOnCurve, priv)
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

	engine := bls12381.NewEngine()
	engine.Reset()
	engine.AddPair(pointOnCurve, publicKey.key)
	leftSide := engine.Result()
	engine.AddPair(sig, engine.G2.One())
	rightSide := engine.Result()
	return leftSide.Equal(rightSide), nil
}

func AggregatePublicKeys(pubKeys []PublicKey) PublicKey {
	g2 := bls12381.NewG2()
	ret := g2.Zero()
	for _, pk := range pubKeys {
		g2.Add(ret, ret, pk.key)
	}
	return NewTrustedPublicKey(ret)
}

func AggregateSignatures(sigs []Signature) Signature {
	g1 := bls12381.NewG1()
	ret := g1.Zero()
	for _, s := range sigs {
		g1.Add(ret, ret, s)
	}
	return ret
}

func VerifyAggregatedSignatureSameMessage(sig Signature, message []byte, pubKeys []PublicKey) (bool, error) {
	return VerifySignature(sig, message, AggregatePublicKeys(pubKeys))
}

func VerifyAggregatedSignatureDifferentMessages(sig Signature, messages [][]byte, pubKeys []PublicKey) (bool, error) {

	if len(messages) != len(pubKeys) {
		return false, errors.New("len(messages) does not match (len(pub keys) in verification")
	}
	engine := bls12381.NewEngine()
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
	engine.AddPair(sig, engine.G2.One())
	rightSide := engine.Result()
	return leftSide.Equal(rightSide), nil
}

// This hashes a message to a [32]byte, then maps the result to the G1 curve using
// the Simplified Shallue-van de Woestijne-Ulas Method, described in Section 6.6.2 of
// https://tools.ietf.org/html/draft-irtf-cfrg-hash-to-curve-06
//
// If keyValidationMode is true, this uses a tweaked version of the padding,
// so that the result will not collide with a result generated in an ordinary signature.
func hashToG1Curve(message []byte, keyValidationMode bool) (*bls12381.PointG1, error) {
	var padding [16]byte
	h := crypto.Keccak256(message)
	if keyValidationMode {
		// modify padding, for domain separation
		padding[0] = 1
	}
	g1 := bls12381.NewG1()
	return g1.MapToCurve(append(padding[:], h...))
}

func PublicKeyToBytes(pub PublicKey) []byte {
	g2 := bls12381.NewG2()
	if pub.validityProof == nil {
		return append([]byte{0}, g2.ToBytes(pub.key)...)
	}
	keyBytes := g2.ToBytes(pub.key)
	sigBytes := SignatureToBytes(pub.validityProof)
	if len(sigBytes) > 255 {
		panic("validity proof too large to serialize")
	}
	return append(append([]byte{byte(len(sigBytes))}, sigBytes...), keyBytes...)
}

func PublicKeyFromBytes(in []byte, trustedSource bool) (PublicKey, error) {
	if len(in) == 0 {
		return PublicKey{}, errors.New("tried to deserialize empty public key")
	}
	g2 := bls12381.NewG2()
	proofLen := int(in[0])
	if proofLen == 0 {
		if !trustedSource {
			return PublicKey{}, errors.New("tried to deserialize unvalidated public key from untrusted source")
		}
		key, err := g2.FromBytes(in[1:])
		if err != nil {
			return PublicKey{}, err
		}
		return NewTrustedPublicKey(key), nil
	} else {
		if len(in) < 1+proofLen {
			return PublicKey{}, errors.New("invalid serialized public key")
		}
		g1 := bls12381.NewG1()
		proofBytes := in[1 : 1+proofLen]
		validityProof, err := g1.FromBytes(proofBytes)
		if err != nil {
			return PublicKey{}, err
		}
		keyBytes := in[1+proofLen:]
		key, err := g2.FromBytes(keyBytes)
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
	return bls12381.NewFr().Set(priv).ToBytes()
}

func PrivateKeyFromBytes(in []byte) (PrivateKey, error) {
	return bls12381.NewFr().FromBytes(in), nil
}

func SignatureToBytes(sig Signature) []byte {
	g1 := bls12381.NewG1()
	return g1.ToBytes(sig)
}

func SignatureFromBytes(in []byte) (Signature, error) {
	g1 := bls12381.NewG1()
	return g1.FromBytes(in)
}
