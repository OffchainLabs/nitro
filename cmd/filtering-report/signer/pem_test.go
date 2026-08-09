// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package signer_test

import (
	"bytes"
	"slices"
	"strings"
	"testing"

	"github.com/offchainlabs/nitro/cmd/filtering-report/signer"
	"github.com/offchainlabs/nitro/cmd/filtering-report/signer/signertest"
)

func assertErrorContains(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("expected error containing %q, got: %v", want, err)
	}
}

func TestParseCombinedPEM_RejectsMismatchedKeyAndCert(t *testing.T) {
	pki := signertest.NewPKI(t)
	priv1, _ := pki.IssueLeaf(t, signertest.DefaultLeafOptions(signertest.DefaultTestSAN))
	_, leafDER2 := pki.IssueLeaf(t, signertest.DefaultLeafOptions(signertest.DefaultTestSAN))

	keyPEM, certPEM := signertest.EncodePEMBundle(t, priv1, leafDER2)
	bundle := slices.Concat(keyPEM, certPEM)

	_, err := signer.ParseCombinedPEM(bundle)
	assertErrorContains(t, err, "private key does not match leaf certificate public key")
}

func TestParseCombinedPEM_RejectsKeyOnly(t *testing.T) {
	pki := signertest.NewPKI(t)
	priv, leafDER := pki.IssueLeaf(t, signertest.DefaultLeafOptions(signertest.DefaultTestSAN))
	keyPEM, _ := signertest.EncodePEMBundle(t, priv, leafDER)

	_, err := signer.ParseCombinedPEM(keyPEM)
	assertErrorContains(t, err, "no CERTIFICATE block found in PEM")
}

func TestParseCombinedPEM_RejectsCertOnly(t *testing.T) {
	pki := signertest.NewPKI(t)
	priv, leafDER := pki.IssueLeaf(t, signertest.DefaultLeafOptions(signertest.DefaultTestSAN))
	_, certPEM := signertest.EncodePEMBundle(t, priv, leafDER)

	_, err := signer.ParseCombinedPEM(certPEM)
	assertErrorContains(t, err, "no PRIVATE KEY block found in PEM")
}

func TestParseCombinedPEM_RejectsDuplicatePrivateKey(t *testing.T) {
	pki := signertest.NewPKI(t)
	priv, leafDER := pki.IssueLeaf(t, signertest.DefaultLeafOptions(signertest.DefaultTestSAN))
	keyPEM, certPEM := signertest.EncodePEMBundle(t, priv, leafDER)
	bundle := slices.Concat(keyPEM, keyPEM, certPEM)

	_, err := signer.ParseCombinedPEM(bundle)
	assertErrorContains(t, err, "PEM contains more than one PRIVATE KEY block")
}

func TestParseCombinedPEM_RejectsCAAsLeaf(t *testing.T) {
	pki := signertest.NewPKI(t)
	keyPEM, certPEM := signertest.EncodePEMBundle(t, pki.CAPriv, pki.CACert.Raw)
	bundle := slices.Concat(keyPEM, certPEM)

	_, err := signer.ParseCombinedPEM(bundle)
	assertErrorContains(t, err, "first certificate in PEM is a CA, expected leaf")
}

func TestParseCombinedPEM_AcceptsLeafFollowedByCAChain(t *testing.T) {
	pki := signertest.NewPKI(t)
	priv, leafDER := pki.IssueLeaf(t, signertest.DefaultLeafOptions(signertest.DefaultTestSAN))
	keyPEM, leafCertPEM := signertest.EncodePEMBundle(t, priv, leafDER)
	bundle := slices.Concat(keyPEM, leafCertPEM, pki.CACertPEM())

	creds, err := signer.ParseCombinedPEM(bundle)
	if err != nil {
		t.Fatalf("expected leaf+chain bundle to parse, got: %v", err)
	}
	if !bytes.Equal(creds.LeafCert().Raw, leafDER) {
		t.Fatal("expected leaf cert to be the first CERTIFICATE block, not the CA")
	}
}

func TestParseCombinedPEM_RejectsUnsupportedBlockType(t *testing.T) {
	bundle := []byte("-----BEGIN EC PRIVATE KEY-----\nQUJD\n-----END EC PRIVATE KEY-----\n")
	_, err := signer.ParseCombinedPEM(bundle)
	assertErrorContains(t, err, "unsupported PEM block type")
}
