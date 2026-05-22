// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

// Package signertest provides test helpers for building an in-memory Ed25519 PKI.
package signertest

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/offchainlabs/nitro/cmd/filtering-report/signer"
)

const DefaultTestSAN = "https://test-webhook-signer.internal"

type PKI struct {
	CACert *x509.Certificate
	CAPriv ed25519.PrivateKey
}

func NewPKI(t *testing.T) *PKI {
	t.Helper()
	caPub, caPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate CA key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, caPub, caPriv)
	if err != nil {
		t.Fatalf("create CA cert: %v", err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("parse CA cert: %v", err)
	}
	return &PKI{CACert: cert, CAPriv: caPriv}
}

func (p *PKI) CACertPEM() []byte {
	return pem.EncodeToMemory(&pem.Block{Type: signer.PEMBlockTypeCertificate, Bytes: p.CACert.Raw})
}

type LeafOptions struct {
	URI       string
	NotBefore time.Time
	NotAfter  time.Time
	KeyUsage  x509.KeyUsage
}

func DefaultLeafOptions(uri string) LeafOptions {
	return LeafOptions{
		URI:       uri,
		NotBefore: time.Now().Add(-time.Hour),
		NotAfter:  time.Now().Add(time.Hour),
		KeyUsage:  x509.KeyUsageDigitalSignature,
	}
}

func (p *PKI) IssueLeaf(t *testing.T, opts LeafOptions) (priv ed25519.PrivateKey, leafDER []byte) {
	t.Helper()
	leafPub, leafPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate leaf key: %v", err)
	}
	uri, err := url.Parse(opts.URI)
	if err != nil {
		t.Fatalf("parse leaf URI: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "test-leaf"},
		NotBefore:    opts.NotBefore,
		NotAfter:     opts.NotAfter,
		KeyUsage:     opts.KeyUsage,
		URIs:         []*url.URL{uri},
	}
	leafDER, err = x509.CreateCertificate(rand.Reader, tmpl, p.CACert, leafPub, p.CAPriv)
	if err != nil {
		t.Fatalf("create leaf cert: %v", err)
	}
	return leafPriv, leafDER
}

func EncodePEMBundle(t *testing.T, priv ed25519.PrivateKey, leafDER []byte) (keyPEM, certPEM []byte) {
	t.Helper()
	keyDER, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		t.Fatalf("marshal PKCS8: %v", err)
	}
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: signer.PEMBlockTypePrivateKey, Bytes: keyDER})
	certPEM = pem.EncodeToMemory(&pem.Block{Type: signer.PEMBlockTypeCertificate, Bytes: leafDER})
	return keyPEM, certPEM
}

func WriteCombinedPEM(t *testing.T, dir string, priv ed25519.PrivateKey, leafDER []byte) string {
	t.Helper()
	keyPEM, certPEM := EncodePEMBundle(t, priv, leafDER)
	path := filepath.Join(dir, "combined.pem")
	if err := os.WriteFile(path, append(keyPEM, certPEM...), 0o600); err != nil {
		t.Fatalf("write PEM file: %v", err)
	}
	return path
}

func WriteCAPEMFile(t *testing.T, dir string, caPEM []byte) string {
	t.Helper()
	path := filepath.Join(dir, "ca.pem")
	if err := os.WriteFile(path, caPEM, 0o600); err != nil {
		t.Fatalf("write CA PEM: %v", err)
	}
	return path
}

func SigningFixture(t *testing.T, opts LeafOptions) (pemPath, caPath string) {
	t.Helper()
	pki := NewPKI(t)
	leafPriv, leafDER := pki.IssueLeaf(t, opts)
	dir := t.TempDir()
	return WriteCombinedPEM(t, dir, leafPriv, leafDER), WriteCAPEMFile(t, dir, pki.CACertPEM())
}
