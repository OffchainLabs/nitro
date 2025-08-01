package arbnode

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	flag "github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/util/stopwaiter"
)

var (
	StateUnmatchedErr = errors.New("state unmatched")
)

type StateCheckerConfig struct {
	PollingInterval        time.Duration `koanf:"polling-interval"`
	ErrorToleranceDuration time.Duration `koanf:"error-tolerance-duration"`

	// http endpoint of the trusted node
	TrustedNodeUrl string `koanf:"trusted-node-url"`
}

var DefaultStateCheckerConfig = StateCheckerConfig{
	PollingInterval:        time.Second * 100,
	ErrorToleranceDuration: time.Minute * 10,
}

func EspressoStateCheckerConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Duration(prefix+".polling-interval", DefaultStateCheckerConfig.PollingInterval, "time after a success")
	f.Duration(prefix+".error-tolerance-duration", DefaultStateCheckerConfig.ErrorToleranceDuration, "error tolerance duration")
	f.String(prefix+".trusted-node-url", DefaultStateCheckerConfig.TrustedNodeUrl, "http endpoint of the trusted node")
}

type StateChecker struct {
	stopwaiter.StopWaiter

	config       StateCheckerConfig
	fatalErrChan chan error

	trustedClient *ethclient.Client
	myClient      *ethclient.Client
}

func NewStateChecker(
	config StateCheckerConfig,
	httpPort int,
	fatalErrChan chan error,
) *StateChecker {
	if config.TrustedNodeUrl == "" {
		log.Warn("trusted node url is empty, state checker will not start")
		return nil
	}

	client, err := ethclient.DialContext(context.Background(), config.TrustedNodeUrl)
	if err != nil {
		panic(err)
	}
	myUrl := fmt.Sprintf("http://localhost:%d", httpPort)
	myClient, err := ethclient.DialContext(context.Background(), myUrl)
	if err != nil {
		log.Warn("failed to dial my node for state checker, state checker will not start", "err", err)
		return nil
	}

	return &StateChecker{
		config:        config,
		fatalErrChan:  fatalErrChan,
		myClient:      myClient,
		trustedClient: client,
	}
}

func (s *StateChecker) Start(ctx context.Context) error {
	s.StopWaiter.Start(ctx, s)

	return s.StartMonitoring(ctx)
}

func (s *StateChecker) StartMonitoring(ctx context.Context) error {
	var firstErrFound time.Time
	return s.CallIterativelySafe(func(ctx context.Context) time.Duration {
		err := s.checkState(ctx)
		if err == nil {
			firstErrFound = time.Time{}
			return s.config.PollingInterval
		}
		if strings.Contains(err.Error(), StateUnmatchedErr.Error()) {
			log.Error("shutting down due to state unmatched", "err", err)
			s.fatalErrChan <- err
			return 0
		}
		if firstErrFound.IsZero() {
			firstErrFound = time.Now()
		} else if time.Since(firstErrFound) > s.config.ErrorToleranceDuration {
			log.Error("shutting down due to error tolerance duration exceeded", "err", err)
			s.fatalErrChan <- err
		} else if strings.Contains(err.Error(), "connection refused") && strings.Contains(err.Error(), "my node") {
			// This case is for the situation where the node haven't started yet
			// returns zero to make sure the state checker checks the state first
			// before the caff node consuming any messages.
			//
			// And if this error lasts for too long, the state checker will shut down
			log.Error("error checking state", "err", err)
			return 0
		}

		log.Error("error checking state", "err", err)
		return s.config.PollingInterval
	})
}

func (s *StateChecker) checkState(ctx context.Context) error {
	block, err := s.trustedClient.BlockByNumber(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to get latest block through trusted node: %w", err)
	}
	blockNumber := block.Number()
	myLatestBlock, err := s.myClient.BlockByNumber(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to get latest block through my node: %w", err)
	}
	myLatestBlockNumber := myLatestBlock.Number()

	if myLatestBlockNumber.Cmp(blockNumber) < 0 {
		log.Info("my node is behind the trusted node", "myBlockNumber", myLatestBlockNumber, "trustedBlockNumber", blockNumber)
		block, err = s.trustedClient.BlockByNumber(ctx, myLatestBlockNumber)
		if err != nil {
			return fmt.Errorf("failed to get block by number through trusted node: %w", err)
		}
		blockNumber = myLatestBlockNumber
	}
	myBlock, err := s.myClient.BlockByNumber(ctx, blockNumber)
	if err != nil {
		return fmt.Errorf("failed to get block by number through my node: %w", err)
	}

	if block.Hash() != myBlock.Hash() {
		err := fmt.Errorf("%s: trusted node: %s, my node: %s", StateUnmatchedErr.Error(), block.Hash(), myBlock.Hash())
		return err
	}
	return nil
}
