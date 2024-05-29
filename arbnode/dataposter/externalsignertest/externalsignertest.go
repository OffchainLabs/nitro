package externalsignertest

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"math/big"
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

	SignerPort   = 1234
	SignerURL    = fmt.Sprintf("https://localhost:%v", SignerPort)
	SignerMethod = "test_signTransaction"
)

type CertAbsPaths struct {
	ServerCert string
	ServerKey  string
	ClientCert string
	ClientKey  string
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

func NewServer(t *testing.T) (*http.Server, *SignerAPI) {
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

	httpServer := &http.Server{
		Addr:              fmt.Sprintf(":%d", SignerPort),
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

	return httpServer, s
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
	tx, err := req.ToTransaction()
	if err != nil {
		return nil, fmt.Errorf("converting transaction arguments into transaction: %w", err)
	}
	signedTx, err := a.SignerFn(a.Address, tx)
	if err != nil {
		return nil, fmt.Errorf("signing transaction: %w", err)
	}
	signedTxBytes, err := signedTx.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("marshaling signed transaction: %w", err)
	}
	return signedTxBytes, nil
}
