// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package signer

import (
	"crypto/ed25519"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
)

const (
	PEMBlockTypePrivateKey  = "PRIVATE KEY"
	PEMBlockTypeCertificate = "CERTIFICATE"
)

func ParseCombinedPEM(data []byte) (*credentials, error) {
	var privateKey ed25519.PrivateKey
	var leaf *x509.Certificate

	rest := data
	for {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			break
		}
		switch block.Type {
		case PEMBlockTypePrivateKey:
			if privateKey != nil {
				return nil, errors.New("PEM contains more than one PRIVATE KEY block")
			}
			key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
			if err != nil {
				return nil, fmt.Errorf("parse PKCS#8 private key: %w", err)
			}
			ed, ok := key.(ed25519.PrivateKey)
			if !ok {
				return nil, fmt.Errorf("private key is not Ed25519 (got %T)", key)
			}
			privateKey = ed
		case PEMBlockTypeCertificate:
			// First CERTIFICATE block is the leaf. Subsequent blocks (e.g. CA chain
			// from cert-manager's CombinedPEM output) are intentionally ignored:
			// only the leaf is included in the X-Signature-Cert header, no intermediates.
			if leaf != nil {
				continue
			}
			cert, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				return nil, fmt.Errorf("parse certificate: %w", err)
			}
			leaf = cert
		default:
			return nil, fmt.Errorf("unsupported PEM block type %q (expected PRIVATE KEY in PKCS#8 form and CERTIFICATE)", block.Type)
		}
	}

	if privateKey == nil {
		return nil, errors.New("no PRIVATE KEY block found in PEM")
	}
	if leaf == nil {
		return nil, errors.New("no CERTIFICATE block found in PEM")
	}
	if leaf.IsCA {
		return nil, errors.New("first certificate in PEM is a CA, expected leaf")
	}

	leafPub, ok := leaf.PublicKey.(ed25519.PublicKey)
	if !ok {
		return nil, fmt.Errorf("leaf certificate public key is not Ed25519 (got %T)", leaf.PublicKey)
	}
	if !leafPub.Equal(privateKey.Public()) {
		return nil, errors.New("private key does not match leaf certificate public key")
	}

	return &credentials{
		privateKey: privateKey,
		leafCert:   leaf,
	}, nil
}

func (c *credentials) LeafCert() *x509.Certificate { return c.leafCert }
