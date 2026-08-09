// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package signertest

import (
	"crypto/ed25519"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strconv"
	"time"

	"github.com/offchainlabs/nitro/cmd/filtering-report/signer"
)

const DefaultTimestampSkew = 5 * time.Minute

type VerifierConfig struct {
	CARootPEMFile string
	ExpectedSAN   string
	TimestampSkew time.Duration
	Now           func() time.Time
}

type Verifier struct {
	rootPool      *x509.CertPool
	expectedSAN   string
	timestampSkew time.Duration
	now           func() time.Time
}

func NewVerifier(c *VerifierConfig) (*Verifier, error) {
	if c == nil {
		return nil, errors.New("config must not be nil")
	}
	if c.CARootPEMFile == "" {
		return nil, errors.New("ca-root-pem-file is required")
	}
	if c.ExpectedSAN == "" {
		return nil, errors.New("expected-san is required")
	}
	rootBytes, err := os.ReadFile(c.CARootPEMFile)
	if err != nil {
		return nil, fmt.Errorf("read CA root PEM: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(rootBytes) {
		return nil, errors.New("CA root PEM contains no valid certificates")
	}
	if c.TimestampSkew < 0 {
		return nil, fmt.Errorf("timestamp-skew must be >= 0, got %s", c.TimestampSkew)
	}
	now := c.Now
	if now == nil {
		now = time.Now
	}
	return &Verifier{
		rootPool:      pool,
		expectedSAN:   c.ExpectedSAN,
		timestampSkew: c.TimestampSkew,
		now:           now,
	}, nil
}

func (v *Verifier) VerifyHTTPRequest(req *http.Request, rawBody []byte) error {
	sigHeader := req.Header.Get(signer.HeaderSignature)
	certHeader := req.Header.Get(signer.HeaderSignatureCert)
	tsHeader := req.Header.Get(signer.HeaderSignatureTimestamp)
	if sigHeader == "" || certHeader == "" || tsHeader == "" {
		return errors.New("missing signature headers")
	}

	signature, err := base64.StdEncoding.DecodeString(sigHeader)
	if err != nil {
		return fmt.Errorf("decode signature: %w", err)
	}
	if len(signature) != ed25519.SignatureSize {
		return fmt.Errorf("signature has wrong length %d, want %d", len(signature), ed25519.SignatureSize)
	}
	certDER, err := base64.StdEncoding.DecodeString(certHeader)
	if err != nil {
		return fmt.Errorf("decode certificate: %w", err)
	}
	leaf, err := x509.ParseCertificate(certDER)
	if err != nil {
		return fmt.Errorf("parse leaf certificate: %w", err)
	}
	if leaf.IsCA {
		return errors.New("presented certificate is a CA, expected leaf")
	}

	now := v.now()
	if _, err := leaf.Verify(x509.VerifyOptions{
		Roots:       v.rootPool,
		CurrentTime: now,
		KeyUsages:   []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
	}); err != nil {
		return fmt.Errorf("verify chain: %w", err)
	}

	if leaf.KeyUsage&x509.KeyUsageDigitalSignature == 0 {
		return errors.New("leaf certificate missing DigitalSignature key usage")
	}

	if !slices.ContainsFunc(leaf.URIs, func(u *url.URL) bool { return u.String() == v.expectedSAN }) {
		return fmt.Errorf("leaf certificate SAN does not contain expected URI %q", v.expectedSAN)
	}

	tsSeconds, err := strconv.ParseInt(tsHeader, 10, 64)
	if err != nil {
		return fmt.Errorf("parse timestamp: %w", err)
	}
	signedAt := time.Unix(tsSeconds, 0)
	if delta := now.Sub(signedAt); delta < -v.timestampSkew || delta > v.timestampSkew {
		return fmt.Errorf("timestamp outside tolerance: delta=%s skew=%s", delta, v.timestampSkew)
	}

	publicKey, ok := leaf.PublicKey.(ed25519.PublicKey)
	if !ok {
		return fmt.Errorf("leaf certificate public key is not Ed25519 (got %T)", leaf.PublicKey)
	}
	payload := signer.BuildSigningPayload(tsHeader, rawBody)
	if !ed25519.Verify(publicKey, payload, signature) {
		return errors.New("signature verification failed")
	}
	return nil
}
