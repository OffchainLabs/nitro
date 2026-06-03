// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package signer

import (
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"

	"github.com/offchainlabs/nitro/util/stopwaiter"
)

const (
	HeaderSignature          = "X-Signature"
	HeaderSignatureCert      = "X-Signature-Cert"
	HeaderSignatureTimestamp = "X-Signature-Timestamp"
)

var reloadFailuresCounter = metrics.NewRegisteredCounter("arb/filtering-report/signer/reload_failures_total", nil)

type Config struct {
	PEMFile        string        `koanf:"pem-file"`
	ReloadInterval time.Duration `koanf:"reload-interval"`
}

var DefaultConfig = Config{
	PEMFile:        "",
	ReloadInterval: time.Hour,
}

func ConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.String(prefix+".pem-file", DefaultConfig.PEMFile, "path to combined PEM file containing Ed25519 private key and leaf certificate")
	f.Duration(prefix+".reload-interval", DefaultConfig.ReloadInterval, "interval between PEM file reloads")
}

func (c *Config) Validate() error {
	if c.PEMFile == "" {
		return errors.New("pem-file is required")
	}
	if c.ReloadInterval <= 0 {
		return fmt.Errorf("reload-interval must be > 0, got %s", c.ReloadInterval)
	}
	return nil
}

type credentials struct {
	privateKey ed25519.PrivateKey
	leafCert   *x509.Certificate
}

type Signer struct {
	stopwaiter.StopWaiter
	pemFile        string
	reloadInterval time.Duration
	creds          atomic.Pointer[credentials]
}

func NewSigner(config *Config) (*Signer, error) {
	if config == nil {
		return nil, errors.New("config must not be nil")
	}
	if err := config.Validate(); err != nil {
		return nil, err
	}
	s := &Signer{pemFile: config.PEMFile, reloadInterval: config.ReloadInterval}
	if err := s.Reload(); err != nil {
		return nil, fmt.Errorf("initial PEM load: %w", err)
	}
	return s, nil
}

func (s *Signer) Start(ctx context.Context) {
	s.StopWaiter.Start(ctx, s)
	s.CallIteratively(func(_ context.Context) time.Duration {
		if err := s.Reload(); err != nil {
			reloadFailuresCounter.Inc(1)
			log.Error("Failed to reload signing PEM, retaining previous credentials", "err", err, "file", s.pemFile)
		} else {
			log.Info("Reloaded signing PEM", "file", s.pemFile)
		}
		return s.reloadInterval
	})
}

func (s *Signer) LeafCert() *x509.Certificate {
	return s.creds.Load().leafCert
}

func (s *Signer) Reload() error {
	data, err := os.ReadFile(s.pemFile)
	if err != nil {
		return fmt.Errorf("read PEM file %q: %w", s.pemFile, err)
	}
	creds, err := ParseCombinedPEM(data)
	if err != nil {
		return err
	}
	if err := checkLeafValidity(creds.leafCert, time.Now(), s.reloadInterval); err != nil {
		return err
	}
	s.creds.Store(creds)
	return nil
}

func checkLeafValidity(leaf *x509.Certificate, now time.Time, reloadInterval time.Duration) error {
	if now.Before(leaf.NotBefore) {
		return fmt.Errorf("leaf certificate not yet valid (NotBefore=%s, now=%s)", leaf.NotBefore, now)
	}
	if now.Add(reloadInterval).After(leaf.NotAfter) {
		return fmt.Errorf("leaf certificate expires within reload interval (NotAfter=%s, now=%s, reload-interval=%s)", leaf.NotAfter, now, reloadInterval)
	}
	return nil
}

func (s *Signer) SignHTTPRequest(req *http.Request, body []byte, now time.Time) {
	creds := s.creds.Load()
	timestamp := strconv.FormatInt(now.Unix(), 10)
	payload := BuildSigningPayload(timestamp, body)
	signature := ed25519.Sign(creds.privateKey, payload)

	req.Header.Set(HeaderSignature, base64.StdEncoding.EncodeToString(signature))
	req.Header.Set(HeaderSignatureCert, base64.StdEncoding.EncodeToString(creds.leafCert.Raw))
	req.Header.Set(HeaderSignatureTimestamp, timestamp)
}

func BuildSigningPayload(timestamp string, body []byte) []byte {
	payload := make([]byte, 0, len(timestamp)+1+len(body))
	payload = append(payload, timestamp...)
	payload = append(payload, '.')
	payload = append(payload, body...)
	return payload
}
