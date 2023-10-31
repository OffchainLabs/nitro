package dataposter

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/google/go-cmp/cmp"
)

func TestParseReplacementTimes(t *testing.T) {
	for _, tc := range []struct {
		desc, replacementTimes string
		want                   []time.Duration
		wantErr                bool
	}{
		{
			desc:             "valid case",
			replacementTimes: "1s,2s,1m,5m",
			want: []time.Duration{
				time.Duration(time.Second),
				time.Duration(2 * time.Second),
				time.Duration(time.Minute),
				time.Duration(5 * time.Minute),
				time.Duration(time.Hour * 24 * 365 * 10),
			},
		},
		{
			desc:             "non-increasing replacement times",
			replacementTimes: "1s,2s,1m,5m,1s",
			wantErr:          true,
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			got, err := parseReplacementTimes(tc.replacementTimes)
			if gotErr := (err != nil); gotErr != tc.wantErr {
				t.Fatalf("Got error: %t, want: %t", gotErr, tc.wantErr)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("parseReplacementTimes(%s) unexpected diff:\n%s", tc.replacementTimes, diff)
			}
		})
	}
}

func TestExternalSigner(t *testing.T) {
	ctx := context.Background()
	httpSrv, srv := newServer(ctx, t)
	t.Cleanup(func() {
		if err := httpSrv.Shutdown(ctx); err != nil {
			t.Fatalf("Error shutting down http server: %v", err)
		}
	})
	cert, key := "./testdata/localhost.crt", "./testdata/localhost.key"
	go func() {
		fmt.Println("Server is listening on port 1234...")
		if err := httpSrv.ListenAndServeTLS(cert, key); err != nil && err != http.ErrServerClosed {
			t.Errorf("ListenAndServeTLS() unexpected error:  %v", err)
			return
		}
	}()
	signer, addr, err := externalSigner(ctx,
		&ExternalSignerCfg{
			Address:          srv.address.Hex(),
			URL:              "https://localhost:1234",
			Method:           "test_signTransaction",
			RootCA:           cert,
			ClientCert:       "./testdata/client.crt",
			ClientPrivateKey: "./testdata/client.key",
		})
	if err != nil {
		t.Fatalf("Error getting external signer: %v", err)
	}
	tx := types.NewTransaction(13, common.HexToAddress("0x01"), big.NewInt(1), 2, big.NewInt(3), []byte{0x01, 0x02, 0x03})
	got, err := signer(ctx, addr, tx)
	if err != nil {
		t.Fatalf("Error signing transaction with external signer: %v", err)
	}
	want, err := srv.signerFn(addr, tx)
	if err != nil {
		t.Fatalf("Error signing transaction: %v", err)
	}
	if diff := cmp.Diff(want.Hash(), got.Hash()); diff != "" {
		t.Errorf("Signing transaction: unexpected diff: %v\n", diff)
	}
}

type server struct {
	handlers map[string]func(*json.RawMessage) (string, error)
	signerFn bind.SignerFn
	address  common.Address
}

type request struct {
	ID     *json.RawMessage `json:"id"`
	Method string           `json:"method"`
	Params *json.RawMessage `json:"params"`
}

type response struct {
	ID     *json.RawMessage `json:"id"`
	Result string           `json:"result,omitempty"`
}

// newServer returns http server and server struct that implements RPC methods.
// It sets up an account in temporary directory and cleans up after test is
// done.
func newServer(ctx context.Context, t *testing.T) (*http.Server, *server) {
	t.Helper()
	signer, address, err := setupAccount("/tmp/keystore")
	if err != nil {
		t.Fatalf("Error setting up account: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll("/tmp/keystore") })

	s := &server{signerFn: signer, address: address}
	s.handlers = map[string]func(*json.RawMessage) (string, error){
		"test_signTransaction": s.signTransaction,
	}
	m := http.NewServeMux()

	clientCert, err := os.ReadFile("./testdata/client.crt")
	if err != nil {
		t.Fatalf("Error reading client certificate: %v", err)
	}
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(clientCert)

	httpSrv := &http.Server{
		Addr:        ":1234",
		Handler:     m,
		ReadTimeout: 5 * time.Second,
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
			ClientAuth: tls.RequireAndVerifyClientCert,
			ClientCAs:  pool,
		},
	}
	m.HandleFunc("/", s.mux)
	return httpSrv, s
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
	txOpts, err := bind.NewKeyStoreTransactorWithChainID(ks, a, big.NewInt(1))
	if err != nil {
		return nil, common.Address{}, fmt.Errorf("creating transactor: %w", err)
	}
	return txOpts.Signer, a.Address, nil
}

// UnmarshallFirst unmarshalls slice of params and returns the first one.
// Parameters in Go ethereum RPC calls are marashalled as slices. E.g.
// eth_sendRawTransaction or eth_signTransaction, marshall transaction as a
// slice of transactions in a message:
// https://github.com/ethereum/go-ethereum/blob/0004c6b229b787281760b14fb9460ffd9c2496f1/rpc/client.go#L548
func unmarshallFirst(params []byte) (*types.Transaction, error) {
	var arr []apitypes.SendTxArgs
	if err := json.Unmarshal(params, &arr); err != nil {
		return nil, fmt.Errorf("unmarshaling first param: %w", err)
	}
	if len(arr) != 1 {
		return nil, fmt.Errorf("argument should be a single transaction, but got: %d", len(arr))
	}
	return arr[0].ToTransaction(), nil
}

func (s *server) signTransaction(params *json.RawMessage) (string, error) {
	tx, err := unmarshallFirst(*params)
	if err != nil {
		return "", err
	}
	signedTx, err := s.signerFn(s.address, tx)
	if err != nil {
		return "", fmt.Errorf("signing transaction: %w", err)
	}
	data, err := rlp.EncodeToBytes(signedTx)
	if err != nil {
		return "", fmt.Errorf("rlp encoding transaction: %w", err)
	}
	return hexutil.Encode(data), nil
}

func (s *server) mux(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "can't read body", http.StatusBadRequest)
		return
	}
	var req request
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "can't unmarshal JSON request", http.StatusBadRequest)
		return
	}
	method, ok := s.handlers[req.Method]
	if !ok {
		http.Error(w, "method not found", http.StatusNotFound)
		return
	}
	result, err := method(req.Params)
	if err != nil {
		fmt.Printf("error calling method: %v\n", err)
		http.Error(w, "error calling method", http.StatusInternalServerError)
		return
	}
	resp := response{ID: req.ID, Result: result}
	respBytes, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, fmt.Sprintf("error encoding response: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(respBytes); err != nil {
		fmt.Printf("error writing response: %v\n", err)
	}
}
