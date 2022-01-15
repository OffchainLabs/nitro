//
// Copyright 2022, Offchain Labs, Inc. All rights reserved.
//

package blsSignatures

import (
	cryptorand "crypto/rand"
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

type PublicKey *bls12381.PointG2

type PrivateKey *big.Int

type Signature *bls12381.PointG1

func GenerateKeys() (PublicKey, PrivateKey, error) {
	privateKey, err := cryptorand.Int(cryptorand.Reader, blsState.g2.Q())
	if err != nil {
		return nil, nil, err
	}
	publicKey := &bls12381.PointG2{}
	blsState.g2.MulScalar(publicKey, blsState.g2.One(), privateKey)
	return publicKey, privateKey, nil
}

// Use for testing only.
func InsecureDeterministicGenerateKeys(seed *big.Int) (PublicKey, PrivateKey, error) {
	privateKey := seed
	publicKey := &bls12381.PointG2{}
	blsState.g2.MulScalar(publicKey, blsState.g2.One(), privateKey)
	return publicKey, privateKey, nil
}

func SignMessage(priv PrivateKey, message []byte) (Signature, error) {
	pointOnCurve, err := hashToG1Curve(message)
	if err != nil {
		return nil, err
	}
	result := &bls12381.PointG1{}
	blsState.g1.MulScalar(result, pointOnCurve, priv)
	return Signature(result), nil
}

func VerifySignature(sig Signature, message []byte, publicKey PublicKey) (bool, error) {
	pointOnCurve, err := hashToG1Curve(message)
	if err != nil {
		return false, err
	}

	engine := blsState.pairingEngine
	engine.Reset()
	engine.AddPair(pointOnCurve, publicKey)
	leftSide := engine.Result()
	engine.AddPair(sig, blsState.g2.One())
	rightSide := engine.Result()
	return leftSide.Equal(rightSide), nil
}

func AggregatePublicKeys(pubKeys []PublicKey) PublicKey {
	ret := blsState.g2.Zero()
	for _, pk := range pubKeys {
		blsState.g2.Add(ret, ret, pk)
	}
	return ret
}

func AggregateSignatures(sigs []Signature) Signature {
	ret := blsState.g1.Zero()
	for _, s := range sigs {
		blsState.g1.Add(ret, ret, s)
	}
	return ret
}

func VerifyAggregatedSignature(sig Signature, message []byte, pubKeys []PublicKey) (bool, error) {
	return VerifySignature(sig, message, AggregatePublicKeys(pubKeys))
}

func hashToG1Curve(message []byte) (*bls12381.PointG1, error) {
	var empty [16]byte
	return blsState.g1.MapToCurve(append(empty[:], crypto.Keccak256(message)...))
}

func PublicKeyToBytes(pub PublicKey) []byte {
	return blsState.g2.ToBytes(pub)
}

func PublicKeyFromBytes(in []byte) (PublicKey, error) {
	return blsState.g2.FromBytes(in)
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
