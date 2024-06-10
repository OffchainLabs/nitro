package externalsignertest

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/offchainlabs/nitro/arbnode/dataposter/externalsigner"
)

var (
	dataPosterPath = "arbnode/dataposter"
	selfPath       = filepath.Join(dataPosterPath, "externalsignertest")
	SignerMethod   = "test_signTransaction"
)

type CertAbsPaths struct {
	ServerCert string
	ServerKey  string
	ClientCert string
	ClientKey  string
}

type SignerServer struct {
	*http.Server
	*SignerAPI
}

func basePath() (string, error) {
	_, file, _, ok := runtime.Caller(1)
	if !ok {
		return "", fmt.Errorf("error getting caller")
	}
	idx := strings.Index(file, selfPath)
	if idx == -1 {
		return "", fmt.Errorf("error determining base path, selfPath: %q is not substring of current file path: %q", selfPath, file)
	}
	return file[:idx], nil
}

func testDataPath() (string, error) {
	base, err := basePath()
	if err != nil {
		return "", fmt.Errorf("getting base path: %w", err)
	}
	return filepath.Join(base, dataPosterPath, "testdata"), nil
}

func CertPaths() (*CertAbsPaths, error) {
	td, err := testDataPath()
	if err != nil {
		return nil, fmt.Errorf("getting test data path: %w", err)
	}
	return &CertAbsPaths{
		ServerCert: filepath.Join(td, "localhost.crt"),
		ServerKey:  filepath.Join(td, "localhost.key"),
		ClientCert: filepath.Join(td, "client.crt"),
		ClientKey:  filepath.Join(td, "client.key"),
	}, nil
}

func NewServer(t *testing.T) *SignerServer {
	rpcServer := rpc.NewServer()
	signer, address, err := setupAccount("/tmp/keystore")
	if err != nil {
		t.Fatalf("Error setting up account: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll("/tmp/keystore") })

	s := &SignerAPI{SignerFn: signer, Address: address}
	if err := rpcServer.RegisterName("test", s); err != nil {
		t.Fatalf("Failed to register EthSigningAPI, error: %v", err)
	}
	cp, err := CertPaths()
	if err != nil {
		t.Fatalf("Error getting certificate paths: %v", err)
	}
	clientCert, err := os.ReadFile(cp.ClientCert)
	if err != nil {
		t.Fatalf("Error reading client certificate: %v", err)
	}
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(clientCert)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Error creating listener: %v", err)
	}
	if err := ln.Close(); err != nil {
		t.Fatalf("Error closing the listener: %v", err)
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

	t.Cleanup(func() {
		if err := httpServer.Close(); err != nil {
			t.Fatalf("Error shutting down http server: %v", err)
		}
	})

	return &SignerServer{httpServer, s}
}

// URL returns the URL of the signer server.
//
// Note: The server must return "localhost" for the hostname part of
// the URL to match the expectations from the TLS certificate.
func (s *SignerServer) URL() string {
	port := strings.Split(s.Addr, ":")[1]
	return fmt.Sprintf("https://localhost:%s", port)
}

func (s *SignerServer) Start() error {
	cp, err := CertPaths()
	if err != nil {
		return err
	}
	if err := s.ListenAndServeTLS(cp.ServerCert, cp.ServerKey); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// setupAccount creates a new account in a given directory, unlocks it, creates
// signer with that account and returns it along with account address.
func setupAccount(dir string) (bind.SignerFn, common.Address, error) {
	ks := keystore.NewKeyStore(
		dir,
		keystore.StandardScryptN,
		keystore.StandardScryptP,
	)
	a, err := ks.NewAccount("password")
	if err != nil {
		return nil, common.Address{}, fmt.Errorf("creating account account: %w", err)
	}
	if err := ks.Unlock(a, "password"); err != nil {
		return nil, common.Address{}, fmt.Errorf("unlocking account: %w", err)
	}
	txOpts, err := bind.NewKeyStoreTransactorWithChainID(ks, a, big.NewInt(1337))
	if err != nil {
		return nil, common.Address{}, fmt.Errorf("creating transactor: %w", err)
	}
	return txOpts.Signer, a.Address, nil
}

type SignerAPI struct {
	SignerFn bind.SignerFn
	Address  common.Address
}

func (a *SignerAPI) SignTransaction(ctx context.Context, req *externalsigner.SignTxArgs) (hexutil.Bytes, error) {
	if req == nil {
		return nil, fmt.Errorf("nil request")
	}
	signedTx, err := a.SignerFn(a.Address, req.ToTransaction())
	if err != nil {
		return nil, fmt.Errorf("signing transaction: %w", err)
	}
	signedTxBytes, err := signedTx.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("marshaling signed transaction: %w", err)
	}
	return signedTxBytes, nil
}
