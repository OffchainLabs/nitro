package main

import (
	"crypto/tls"
	"crypto/x509"
	"math/big"
	"net/http"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/arbnode/dataposter"
	"github.com/offchainlabs/nitro/arbnode/dataposter/externalsignertest"
	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/cmd/util"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func main() {
	args := os.Args
	if len(args) != 2 {
		panic("Usage: mockexternalsigner [private_key]")
	}
	srv, err := NewServer(args[1])
	if err != nil {
		panic(err)
	}
	go func() {
		if err := srv.Start(); err != nil {
			panic(err)
		}
	}()
	signerCfg, err := dataposter.ExternalSignerTestCfg(srv.Address, srv.URL())
	if err != nil {
		panic(err)
	}
	print(" --externalSignerUrl " + signerCfg.URL)
	print(" --externalSignerAddress " + signerCfg.Address)
	print(" --externalSignerMethod " + signerCfg.Method)
	print(" --externalSignerRootCA " + signerCfg.RootCA)
	print(" --externalSignerClientCert " + signerCfg.ClientCert)
	print(" --externalSignerClientPrivateKey " + signerCfg.ClientPrivateKey)
	if signerCfg.InsecureSkipVerify {
		print(" --externalSignerInsecureSkipVerify ")
	}
}

func NewServer(privateKey string) (*externalsignertest.SignerServer, error) {
	rpcServer := rpc.NewServer()
	txOpts, _, err := util.OpenWallet(
		"mockexternalsigner",
		&genericconf.WalletConfig{PrivateKey: privateKey},
		big.NewInt(1337),
	)
	if err != nil {
		return nil, err
	}
	s := &externalsignertest.SignerAPI{SignerFn: txOpts.Signer, Address: txOpts.From}
	if err := rpcServer.RegisterName("test", s); err != nil {
		return nil, err
	}
	cp, err := externalsignertest.CertPaths()
	if err != nil {
		return nil, err
	}
	clientCert, err := os.ReadFile(cp.ClientCert)
	if err != nil {
		return nil, err
	}
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(clientCert)

	ln, err := testhelpers.FreeTCPPortListener()
	if err != nil {
		return nil, err
	}

	httpServer := &http.Server{
		Addr:              ln.Addr().String(),
		Handler:           rpcServer,
		ReadTimeout:       30 * time.Second,
		ReadHeaderTimeout: 30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
			ClientAuth: tls.RequireAndVerifyClientCert,
			ClientCAs:  pool,
		},
	}

	return &externalsignertest.SignerServer{
		Server:    httpServer,
		SignerAPI: s,
		Listener:  ln,
	}, nil
}
