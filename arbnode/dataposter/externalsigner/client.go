// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package externalsigner

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/arbnode/dataposter/config"
)

func rpcClient(ctx context.Context, opts *config.ExternalSignerCfg) (*rpc.Client, error) {
	tlsCfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
		// Dataposter verifies that signed transaction was signed by the account
		// that it expects to be signed with. So signer is already authenticated
		// on application level and does not need to rely on TLS for authentication.
		InsecureSkipVerify: opts.InsecureSkipVerify, // #nosec G402
	}

	if opts.ClientCert != "" && opts.ClientPrivateKey != "" {
		log.Info("Client certificate for external signer is enabled")
		clientCert, err := tls.LoadX509KeyPair(opts.ClientCert, opts.ClientPrivateKey)
		if err != nil {
			return nil, fmt.Errorf("error loading client certificate and private key: %w", err)
		}
		tlsCfg.Certificates = []tls.Certificate{clientCert}
	}

	if opts.RootCA != "" {
		rootCrt, err := os.ReadFile(opts.RootCA)
		if err != nil {
			return nil, fmt.Errorf("error reading external signer root CA: %w", err)
		}
		rootCertPool := x509.NewCertPool()
		rootCertPool.AppendCertsFromPEM(rootCrt)
		tlsCfg.RootCAs = rootCertPool
	}

	return rpc.DialOptions(
		ctx,
		opts.URL,
		rpc.WithHTTPClient(
			&http.Client{
				Transport: &http.Transport{
					TLSClientConfig: tlsCfg,
				},
			},
		),
	)
}
