package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"context"
	"math/big"
	"time"

	"github.com/OffchainLabs/new-rollup-exploration/protocol"
	"github.com/OffchainLabs/new-rollup-exploration/state-manager"
	"github.com/OffchainLabs/new-rollup-exploration/util"
	"github.com/OffchainLabs/new-rollup-exploration/validator"
	"github.com/ethereum/go-ethereum/common"
	// "github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"os"
	"path/filepath"
)

var clients = make(map[*websocket.Conn]bool)
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type server struct {
	chain     protocol.OnChainProtocol
	manager   statemanager.Manager
	validator *validator.Validator
}

func main() {
	ctx := context.Background()
	ref := util.NewRealTimeReference()
	chain := protocol.NewAssertionChain(ctx, ref, time.Minute)

	alice := common.BytesToAddress([]byte{1})

	// Increase the balance for each validator in the test.
	bal := big.NewInt(0).Mul(protocol.Gwei, big.NewInt(100))
	if err := chain.Tx(func(tx *protocol.ActiveTx, p protocol.OnChainProtocol) error {
		chain.AddToBalance(tx, alice, bal)
		return nil
	}); err != nil {
		panic(err)
	}
	stateRoots := make([]common.Hash, 100)
	for i := uint64(0); i < 100; i++ {
		stateRoots[i] = util.HashForUint(i)
	}

	manager := statemanager.New(stateRoots)
	v, err := validator.New(
		ctx,
		chain,
		manager,
		validator.WithName("alice"),
		validator.WithAddress(alice),
		validator.WithDisableLeafCreation(),
		validator.WithTimeReference(ref),
		validator.WithChallengeVertexWakeInterval(time.Second),
	)
	if err != nil {
		panic(err)
	}

	harnessObserver := make(chan protocol.ChallengeEvent, 100)
	chain.SubscribeChallengeEvents(ctx, harnessObserver)

	fmt.Println("Started validator")
	go v.Start(ctx)
	go echo()

	wd, _ := os.Getwd()
	pth := filepath.Join(wd, "web")
	fs := http.FileServer(http.Dir(pth))
	fmt.Println(pth)

	http.Handle("/", fs)
	http.HandleFunc("/api/ws", wsHandler)
	http.HandleFunc("/api/leaf/create", handleConfigUpdate)

	// Render the config.
	s := &srv{
		cfg: &config{
			NumValidators: 1,
		},
	}
	http.HandleFunc("/api/config", s.renderConfig)

	fmt.Println("Server listening on port 8000...")
	log.Fatal(http.ListenAndServe(":8000", nil))
}

type user struct{}

type srv struct {
	cfg *config
}

type config struct {
	NumValidators uint64 `json:"num_validators"`
}

func (s *srv) renderConfig(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(s.cfg)
}

func handleConfigUpdate(w http.ResponseWriter, r *http.Request) {
	// Check the request method
	if r.Method == http.MethodPost {
		// Decode the request body into a User struct
		var user user
		err := json.NewDecoder(r.Body).Decode(&user)
		if err != nil {
			http.Error(w, "Error decoding request body", http.StatusBadRequest)
			return
		}

		// Print the user information to the console
		fmt.Printf("Received user: %+v\n", user)

		// Send a JSON response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(user)
	} else {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
	}
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Handling websocket")
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Handled!")

	// register client
	clients[ws] = true
}

// 3
func echo() {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		<-t.C
		// send to every client that is currently connected
		for client := range clients {
			err := client.WriteMessage(websocket.TextMessage, []byte("hi"))
			if err != nil {
				log.Printf("Websocket error: %s", err)
				client.Close()
				delete(clients, client)
			}
		}
	}
}
