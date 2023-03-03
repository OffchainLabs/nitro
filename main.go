package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"sync"
	"time"

	goimpl "github.com/OffchainLabs/challenge-protocol-v2/protocol/go-implementation"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/OffchainLabs/challenge-protocol-v2/validator"
	"github.com/OffchainLabs/challenge-protocol-v2/web"
	"github.com/ethereum/go-ethereum/common"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"github.com/sirupsen/logrus"
)

var (
	log      = logrus.WithField("prefix", "visualizer")
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)

type config struct {
	NumValidators          uint8    `json:"num_validators"`
	NumStates              uint64   `json:"num_states"`
	DefaultBalance         *big.Int `json:"initial_balance"`
	ChallengePeriodSeconds uint64   `json:"challenge_period_seconds"`
	DivergeHeight          uint64   `json:"disagree_at_height"`
}

func defaultConfig() *config {
	defaultBalance := new(big.Int).Add(goimpl.AssertionStake, goimpl.ChallengeVertexStake)
	return &config{
		NumValidators:          2,
		NumStates:              10,
		DefaultBalance:         defaultBalance,
		ChallengePeriodSeconds: 60,
		DivergeHeight:          3,
	}
}

type server struct {
	lock             sync.RWMutex
	ctx              context.Context
	cancelFn         context.CancelFunc
	cfg              *config
	port             uint
	challengeManager goimpl.ChallengeManagerInterface
	validators       []*validator.Validator
	timeRef          *util.ArtificialTimeReference
	wsClients        map[*websocket.Conn]bool
}

func (s *server) renderConfig(c echo.Context) error {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return c.JSON(http.StatusOK, s.cfg)
}

func (s *server) updateConfig(c echo.Context) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	req := defaultConfig()
	defer c.Request().Body.Close()
	enc, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(enc, req); err != nil {
		return err
	}

	log.Info("Received update config request, restarting application...")
	// Cancel the current runtime of the application, wait a bit for cleanup,
	// then restart the application with the updated configuration.
	s.cancelFn()

	time.Sleep(2 * time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	s.cancelFn = cancel
	s.ctx = ctx
	s.cfg = req
	go s.startBackgroundRoutines(ctx, s.cfg)

	log.Info("Successfully restarted background routines")

	return c.JSON(http.StatusOK, s.cfg)
}

type assertionCreationRequest struct {
	Index uint8 `json:"index"`
}

func (s *server) triggerAssertionCreation(c echo.Context) error {
	req := &assertionCreationRequest{}
	defer c.Request().Body.Close()
	enc, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(enc, req); err != nil {
		return err
	}
	if int(req.Index) >= len(s.validators) {
		return errors.New("index out of rnage")
	}
	s.lock.RLock()
	v := s.validators[req.Index]
	s.lock.RUnlock()
	assertion, err := v.SubmitLeafCreation(s.ctx)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, assertion)
}

func (s *server) registerWebsocketConnection(c echo.Context) error {
	ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		log.Fatal(err)
	}
	s.lock.Lock()
	s.wsClients[ws] = true
	s.lock.Unlock()
	return nil
}

func (s *server) stepTimeReference(c echo.Context) error {
	s.timeRef.Add(time.Second)
	return c.JSON(http.StatusOK, nil)
}

func (s *server) startBackgroundRoutines(ctx context.Context, cfg *config) {
	s.timeRef = util.NewArtificialTimeReference()
	validators, challengeManager, err := initializeSystem(ctx, s.timeRef, cfg)
	if err != nil {
		panic(err)
	}
	s.validators = validators
	s.challengeManager = challengeManager
	challengeObserver := make(chan goimpl.ChallengeEvent, 100)
	chainObserver := make(chan goimpl.AssertionChainEvent, 100)
	s.challengeManager.SubscribeChallengeEvents(ctx, challengeObserver)
	s.challengeManager.SubscribeChainEvents(ctx, chainObserver)

	go s.sendChainEventsToClients(ctx, challengeObserver, chainObserver)

	for _, v := range validators {
		go v.Start(ctx)
	}
	log.Infof("Started application background routines successfully with config %+v", s.cfg)
}

type event struct {
	Typ       string                `json:"typ"`
	To        string                `json:"to"`
	From      string                `json:"from"`
	BecomesPS bool                  `json:"becomes_ps"`
	Validator string                `json:"validator"`
	Vis       *goimpl.Visualization `json:"vis"`
}

func (s *server) sendChainEventsToClients(
	ctx context.Context,
	chalEvs <-chan goimpl.ChallengeEvent,
	chainEvs <-chan goimpl.AssertionChainEvent,
) {
	for {
		select {
		case ev := <-chalEvs:
			log.Infof("Got challenge event: %+T, and %+v", ev, ev)
			vis := s.challengeManager.Visualize(ctx, &goimpl.ActiveTx{TxStatus: goimpl.ReadOnlyTxStatus})
			s.lock.RLock()
			eventToSend := &event{
				Typ: fmt.Sprintf("%+T", ev),
				Vis: vis,
			}

			switch specificEv := ev.(type) {
			case *goimpl.ChallengeLeafEvent:
				eventToSend.BecomesPS = specificEv.BecomesPS
				eventToSend.Validator = fmt.Sprintf("%x", specificEv.Validator[len(specificEv.Validator)-1:])
			case *goimpl.ChallengeMergeEvent:
				eventToSend.From = fmt.Sprintf("%d", specificEv.FromHistory.Height)
				eventToSend.To = fmt.Sprintf("%d", specificEv.ToHistory.Height)
				eventToSend.BecomesPS = specificEv.BecomesPS
				eventToSend.Validator = fmt.Sprintf("%x", specificEv.Validator[len(specificEv.Validator)-1:])
			case *goimpl.ChallengeBisectEvent:
				eventToSend.From = fmt.Sprintf("%d", specificEv.FromHistory.Height)
				eventToSend.To = fmt.Sprintf("%d", specificEv.ToHistory.Height)
				eventToSend.BecomesPS = specificEv.BecomesPS
				eventToSend.Validator = fmt.Sprintf("%x", specificEv.Validator[len(specificEv.Validator)-1:])
			}

			enc, err := json.Marshal(eventToSend)
			if err != nil {
				log.Error(err)
				continue
			}

			// send to every client that is currently connected
			for client := range s.wsClients {
				err := client.WriteMessage(websocket.TextMessage, enc)
				if err != nil {
					if err = client.Close(); err != nil {
						log.Error(err)
						return
					}
					delete(s.wsClients, client)
				}
			}
			s.lock.RUnlock()
		case ev := <-chainEvs:
			log.Infof("Got chain event: %+T, and %+v", ev, ev)
			vis := s.challengeManager.Visualize(ctx, &goimpl.ActiveTx{TxStatus: goimpl.ReadOnlyTxStatus})
			s.lock.RLock()
			eventToSend := &event{
				Typ: fmt.Sprintf("%+T", ev),
				Vis: vis,
			}
			switch specificEv := ev.(type) {
			case *goimpl.CreateLeafEvent:
				eventToSend.Validator = fmt.Sprintf("%x", specificEv.Validator[len(specificEv.Validator)-1:])
			case *goimpl.StartChallengeEvent:
				eventToSend.Validator = fmt.Sprintf("%x", specificEv.Validator[len(specificEv.Validator)-1:])
			default:
			}
			enc, err := json.Marshal(eventToSend)
			if err != nil {
				log.Error(err)
				continue
			}

			// send to every client that is currently connected
			for client := range s.wsClients {
				err := client.WriteMessage(websocket.TextMessage, enc)
				if err != nil {
					if err = client.Close(); err != nil {
						log.Error(err)
						return
					}
					delete(s.wsClients, client)
				}
			}
			s.lock.RUnlock()
		case <-ctx.Done():
			return
		//nolint:staticcheck
		default:
		}
	}
}

// Registers all of the server's routes for the web application.
func (s *server) registerRoutes(e *echo.Echo) {
	// Register the frontend website static assets including HTML.
	web.RegisterHandlers(e)

	// Handle websocket connection registration.
	e.GET("/api/ws", s.registerWebsocketConnection)

	// Configuration related-handlers, either reading the config
	// or updating the config and restarting the application.
	e.GET("/api/config", s.renderConfig)
	e.POST("/api/config", s.updateConfig)

	// API triggers of validator actions, such as leaf creation at a validator's
	// latest height via the web app.
	e.POST("/api/assertions", s.triggerAssertionCreation)
	e.POST("/api/step", s.stepTimeReference)
}

func initializeSystem(
	ctx context.Context,
	timeRef util.TimeReference,
	cfg *config,
) ([]*validator.Validator, goimpl.ChallengeManagerInterface, error) {
	challengeManager := goimpl.NewAssertionChain(ctx, timeRef, time.Duration(cfg.ChallengePeriodSeconds)*time.Second)

	validatorAddrs := make([]common.Address, cfg.NumValidators)
	for i := uint8(0); i < cfg.NumValidators; i++ {
		// Make the addrs 1-indexed so that we don't use the zero address.
		validatorAddrs[i] = common.BytesToAddress([]byte{i + 1})
	}

	// Increase the balance for each validator in the test.
	bal := big.NewInt(0).Add(goimpl.AssertionStake, goimpl.ChallengeVertexStake)
	err := challengeManager.Tx(func(tx *goimpl.ActiveTx) error {
		for _, addr := range validatorAddrs {
			challengeManager.AddToBalance(tx, addr, bal)
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	// Initialize each validator associated state roots which diverge
	// at specified points in the test config.
	validatorStateRoots := make([][]common.Hash, cfg.NumValidators)
	for i := uint8(0); i < cfg.NumValidators; i++ {
		divergenceHeight := cfg.DivergeHeight
		stateRoots := make([]common.Hash, cfg.NumStates)
		for i := uint64(0); i < cfg.NumStates; i++ {
			if divergenceHeight == 0 || i < divergenceHeight {
				stateRoots[i] = util.HashForUint(i)
			} else {
				divergingRoot := make([]byte, 32)
				_, err = rand.Read(divergingRoot)
				if err != nil {
					return nil, nil, err
				}
				stateRoots[i] = common.BytesToHash(divergingRoot)
			}
		}
		validatorStateRoots[i] = stateRoots
	}

	// Initialize each validator.
	validators := make([]*validator.Validator, cfg.NumValidators)
	for i := 0; i < len(validators); i++ {
		addr := validatorAddrs[i]
		// TODO(RJ): This file is broken and needs to take in real dependencies. Currently uses goimpl but should use solimpl
		v, valErr := validator.New(
			ctx,
			nil,
			nil,
			nil,
			common.Address{},
			validator.WithName(fmt.Sprintf("%d", i)),
			validator.WithAddress(addr),
			validator.WithDisableLeafCreation(),
			validator.WithTimeReference(timeRef),
			validator.WithChallengeVertexWakeInterval(time.Second),
		)
		if valErr != nil {
			return nil, nil, valErr
		}
		validators[i] = v
	}
	return validators, challengeManager, nil
}

// Initializes a server that is able to start validators, trigger
// validator events, and provides data on their events via websocket connections.
func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := defaultConfig()
	s := &server{
		ctx:       ctx,
		cancelFn:  cancel,
		cfg:       cfg,
		port:      8000,
		wsClients: map[*websocket.Conn]bool{},
	}

	e := echo.New()
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogURI:    true,
		LogStatus: true,
		LogValuesFunc: func(c echo.Context, values middleware.RequestLoggerValues) error {
			return nil
		},
	}))

	// Register all the server routes for the application.
	s.registerRoutes(e)

	// Start the main application routines in the background.
	go s.startBackgroundRoutines(ctx, cfg)

	// Listen and serve the web application.
	log.Infof("Server listening on port %d", s.port)
	if err := e.Start(fmt.Sprintf(":%d", s.port)); err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
