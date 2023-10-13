package dataposter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
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

type Args struct {
	Name string
}

func TestRPC(t *testing.T) {
	srv, err := newServer()
	if err != nil {
		fmt.Printf("Erorr creating server: %v", err)
		return
	}
	http.HandleFunc("/", srv.mux)
	go func() {
		fmt.Println("Server is listening on port 1234...")
		t.Errorf("error listening: %v", http.ListenAndServe(":1234", nil))
	}()

	if err != nil {
		t.Fatalf("Error creating a server: %v", err)
	}
	ctx := context.Background()
	signer, addr, err := externalSigner(ctx, srv.address.Hex(), "http://127.0.0.1:1234")
	if err != nil {
		t.Fatalf("Error getting external signer: %v", err)
	}
	tx := types.NewTransaction(0, common.HexToAddress("0x01"), big.NewInt(0), 3000000, big.NewInt(30000), nil)
	signedTx, err := signer(ctx, addr, tx)
	if err != nil {
		t.Errorf("Error signing transaction: %v", err)
	}
	if diff := cmp.Diff(signedTx, tx); diff != "" {
		t.Errorf("diff: %v\n", diff)
	}

}

// func setupServer(t *testing.T) {
// 	t.Skip()
// 	t.Helper()
// 	srv, err := newServer()
// 	if err != nil {
// 		fmt.Printf("Erorr creating server: %v", err)
// 		return
// 	}
// 	http.HandleFunc("/", srv.mux)
// 	fmt.Println("Server is listening on port 1234...")
// 	t.Fatal(http.ListenAndServe(":1234", nil))
// }

type SigningService struct{}

func (h *SigningService) SignTx(r *http.Request, args *Args, reply *string) error {
	*reply = "Hello, " + args.Name + "!"
	return nil
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
	Result string           `json:"result"`
}

func newServer() (*server, error) {
	signer, address, err := setupAccount("/tmp/keystore")
	if err != nil {
		return nil, err
	}
	s := &server{
		signerFn: signer,
		address:  address,
	}
	s.handlers = map[string]func(*json.RawMessage) (string, error){
		"eth_signTransaction": s.signTransaction,
	}
	return s, nil
}

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
	log.Printf("Created account: %s", a.Address.Hex())
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
func unmarshallFirst(params []byte) (any, error) {
	var arr []any
	if err := json.Unmarshal(params, &arr); err != nil {
		return "", fmt.Errorf("unmarshaling first param: %w", err)
	}
	return arr[0], nil
}

// func unmarshallTx(params *json.RawMessage) (*types.Transaction, error) {

// }

func encodeTx(tx *types.Transaction) (string, error) {
	data, err := tx.MarshalBinary()
	if err != nil {
		return "", err
	}
	return hexutil.Encode(data), nil
}

func (s *server) signTransaction(params *json.RawMessage) (string, error) {
	param, err := unmarshallFirst(*params)
	if err != nil {
		return "", err
	}

	fmt.Printf("anodar first parameter: %q\n", param)
	data, err := hexutil.Decode("0xe280827530832dc6c09400000000000000000000000000000000000000018080808080")
	if err != nil {
		return "", fmt.Errorf("decoding hex: %w", err)
	}
	fmt.Printf("decoded data: %v\n", data)
	var tx types.Transaction
	if err := tx.UnmarshalBinary(data); err != nil {
		return "", fmt.Errorf("unmarshaling tx: %w", err)
	}
	fmt.Printf("tx: %v\n", tx)
	return encodeTx(&tx)
	// var txs []*types.Transaction
	// for _, arg := range args {
	// 	signedTx, err := s.signerFn(s.address, arg)
	// 	if err != nil {
	// 		return "", fmt.Errorf("signing transaction: %w", err)
	// 	}
	// 	txs = append(txs, signedTx)
	// }
	// enc, err := rlp.EncodeToBytes(txs)
	// if err != nil {
	// 	return "", fmt.Errorf("encoding transactions to bytes: %w", err)
	// }
	// return hexutil.Encode(enc), nil
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
		http.Error(w, "error encoding response", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(respBytes)
}
