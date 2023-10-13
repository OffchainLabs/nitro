package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

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

func new() (*server, error) {
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

func (s *server) signTransaction(params *json.RawMessage) (string, error) {
	var args []apitypes.SendTxArgs
	if err := json.Unmarshal(*params, &args); err != nil {
		return "", fmt.Errorf("unmarshaling params: %w", err)
	}
	var txs []*types.Transaction
	for _, arg := range args {
		signedTx, err := s.signerFn(s.address, arg.ToTransaction())
		if err != nil {
			return "", fmt.Errorf("signing transaction: %w", err)
		}
		txs = append(txs, signedTx)
	}
	enc, err := rlp.EncodeToBytes(txs)
	if err != nil {
		return "", fmt.Errorf("encoding transactions to bytes: %w", err)
	}
	return hexutil.Encode(enc), nil
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

func main() {
	srv, err := new()
	if err != nil {
		fmt.Printf("Erorr creating server: %w", err)
		return
	}
	http.HandleFunc("/", srv.mux)
	fmt.Println("Server is listening on port 1234...")
	log.Fatal(http.ListenAndServe(":1234", nil))
}
