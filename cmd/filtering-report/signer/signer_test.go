// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package signer_test

import (
	"bytes"
	"os"
	"testing"
	"time"

	"github.com/offchainlabs/nitro/cmd/filtering-report/signer"
	"github.com/offchainlabs/nitro/cmd/filtering-report/signer/signertest"
)

const testReloadInterval = time.Minute

func TestSigner_ReloadPicksUpNewCert(t *testing.T) {
	pki := signertest.NewPKI(t)
	priv1, leaf1 := pki.IssueLeaf(t, signertest.DefaultLeafOptions(signertest.DefaultTestSAN))
	dir := t.TempDir()
	pemPath := signertest.WriteCombinedPEM(t, dir, priv1, leaf1)

	s, err := signer.NewSigner(&signer.Config{PEMFile: pemPath, ReloadInterval: testReloadInterval})
	if err != nil {
		t.Fatalf("NewSigner: %v", err)
	}
	first := s.LeafCert().Raw

	priv2, leaf2 := pki.IssueLeaf(t, signertest.DefaultLeafOptions(signertest.DefaultTestSAN))
	keyPEM, certPEM := signertest.EncodePEMBundle(t, priv2, leaf2)
	if err := os.WriteFile(pemPath, append(keyPEM, certPEM...), 0o600); err != nil {
		t.Fatalf("rewrite PEM: %v", err)
	}
	if err := s.Reload(); err != nil {
		t.Fatalf("reload: %v", err)
	}
	if bytes.Equal(first, s.LeafCert().Raw) {
		t.Fatal("expected leafDER to change after reload")
	}
}

func testReloadKeepsOldOnInvalidLeaf(t *testing.T, modifyOpts func(*signertest.LeafOptions)) {
	t.Helper()
	pki := signertest.NewPKI(t)
	priv1, leaf1 := pki.IssueLeaf(t, signertest.DefaultLeafOptions(signertest.DefaultTestSAN))
	dir := t.TempDir()
	pemPath := signertest.WriteCombinedPEM(t, dir, priv1, leaf1)

	s, err := signer.NewSigner(&signer.Config{PEMFile: pemPath, ReloadInterval: testReloadInterval})
	if err != nil {
		t.Fatalf("NewSigner: %v", err)
	}
	original := s.LeafCert()

	invalidOpts := signertest.DefaultLeafOptions(signertest.DefaultTestSAN)
	modifyOpts(&invalidOpts)
	priv2, leaf2 := pki.IssueLeaf(t, invalidOpts)
	keyPEM, certPEM := signertest.EncodePEMBundle(t, priv2, leaf2)
	if err := os.WriteFile(pemPath, append(keyPEM, certPEM...), 0o600); err != nil {
		t.Fatalf("rewrite PEM: %v", err)
	}
	if err := s.Reload(); err == nil {
		t.Fatal("expected reload to reject invalid leaf")
	}
	if s.LeafCert() != original {
		t.Fatal("expected credentials to be retained when reload sees invalid leaf")
	}
}

func TestSigner_ReloadKeepsOldOnExpiredLeaf(t *testing.T) {
	testReloadKeepsOldOnInvalidLeaf(t, func(opts *signertest.LeafOptions) {
		opts.NotAfter = time.Now().Add(-time.Minute)
	})
}

func TestSigner_ReloadKeepsOldOnNotYetValidLeaf(t *testing.T) {
	testReloadKeepsOldOnInvalidLeaf(t, func(opts *signertest.LeafOptions) {
		opts.NotBefore = time.Now().Add(time.Hour)
		opts.NotAfter = time.Now().Add(2 * time.Hour)
	})
}

func TestSigner_ReloadKeepsOldOnParseError(t *testing.T) {
	pemPath, _ := signertest.SigningFixture(t, signertest.DefaultLeafOptions(signertest.DefaultTestSAN))

	s, err := signer.NewSigner(&signer.Config{PEMFile: pemPath, ReloadInterval: testReloadInterval})
	if err != nil {
		t.Fatalf("NewSigner: %v", err)
	}
	original := s.LeafCert()

	if err := os.WriteFile(pemPath, []byte("not a pem"), 0o600); err != nil {
		t.Fatalf("rewrite PEM: %v", err)
	}
	if err := s.Reload(); err == nil {
		t.Fatal("expected reload error on garbage PEM")
	}
	if s.LeafCert() != original {
		t.Fatal("expected credentials to be retained after parse failure")
	}
}
