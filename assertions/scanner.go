// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE

// Package assertions contains testing utilities for posting and scanning for
// assertions on chain, which are useful for simulating the responsibilities
// of Arbitrum Nitro and initiating challenges as needed using our challenge manager.
package assertions

import (
	"context"
	"crypto/rand"
	"math/big"
	"os"
	"time"

	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	"github.com/OffchainLabs/bold/challenge-manager/types"
	l2stateprovider "github.com/OffchainLabs/bold/layer2-state-provider"
	retry "github.com/OffchainLabs/bold/runtime"
	"github.com/OffchainLabs/bold/solgen/go/rollupgen"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/pkg/errors"
)

var (
	srvlog = log.New("service", "assertions")
)

func init() {
	srvlog.SetHandler(log.StreamHandler(os.Stdout, log.LogfmtFormat()))
}

// Scanner checks for posted, onchain assertions via a polling mechanism since the latest confirmed,
// up to the latest block, and keeps doing so as the chain advances. With each observed assertion,
// it determines whether or not it should challenge it.
type Scanner struct {
	chain                    protocol.AssertionChain
	backend                  bind.ContractBackend
	challengeCreator         types.ChallengeCreator
	challengeReader          types.ChallengeReader
	stateProvider            l2stateprovider.Provider
	pollInterval             time.Duration
	rollupAddr               common.Address
	validatorName            string
	forksDetectedCount       uint64
	challengesSubmittedCount uint64
	assertionsProcessedCount uint64
}

// NewScanner creates a scanner from the required dependencies.
func NewScanner(
	chain protocol.AssertionChain,
	stateProvider l2stateprovider.Provider,
	backend bind.ContractBackend,
	challengeManager types.ChallengeManager,
	rollupAddr common.Address,
	validatorName string,
	pollInterval time.Duration,
) *Scanner {
	return &Scanner{
		chain:                    chain,
		backend:                  backend,
		stateProvider:            stateProvider,
		challengeCreator:         challengeManager,
		challengeReader:          challengeManager,
		rollupAddr:               rollupAddr,
		validatorName:            validatorName,
		pollInterval:             pollInterval,
		forksDetectedCount:       0,
		challengesSubmittedCount: 0,
		assertionsProcessedCount: 0,
	}
}

// Start scanning the blockchain for assertion creation events in a polling manner
// from the latest confirmed assertion.
func (s *Scanner) Start(ctx context.Context) {
	latestConfirmed, err := s.chain.LatestConfirmed(ctx)
	if err != nil {
		srvlog.Error("Could not get latest confirmed assertion", log.Ctx{"err": err})
		return
	}
	fromBlock, err := latestConfirmed.CreatedAtBlock()
	if err != nil {
		srvlog.Error("Could not get creation block", log.Ctx{"err": err})
		return
	}

	filterer, err := retry.UntilSucceeds(ctx, func() (*rollupgen.RollupUserLogicFilterer, error) {
		return rollupgen.NewRollupUserLogicFilterer(s.rollupAddr, s.backend)
	})
	if err != nil {
		srvlog.Error("Could not get rollup user logic filterer", log.Ctx{"err": err})
		return
	}
	ticker := time.NewTicker(s.pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			latestBlock, err := s.backend.HeaderByNumber(ctx, nil)
			if err != nil {
				srvlog.Error("Could not get header by number", log.Ctx{"err": err})
				continue
			}
			if !latestBlock.Number.IsUint64() {
				srvlog.Error("Latest block number was not a uint64")
				continue
			}
			toBlock := latestBlock.Number.Uint64()
			if fromBlock == toBlock {
				continue
			}
			filterOpts := &bind.FilterOpts{
				Start:   fromBlock,
				End:     &toBlock,
				Context: ctx,
			}
			_, err = retry.UntilSucceeds(ctx, func() (bool, error) {
				return true, s.checkForAssertionAdded(ctx, filterer, filterOpts)
			})
			if err != nil {
				srvlog.Error("Could not check for assertion added", log.Ctx{"err": err})
				return
			}
			fromBlock = toBlock
		case <-ctx.Done():
			return
		}
	}
}

func (s *Scanner) checkForAssertionAdded(
	ctx context.Context,
	filterer *rollupgen.RollupUserLogicFilterer,
	filterOpts *bind.FilterOpts,
) error {
	it, err := filterer.FilterAssertionCreated(filterOpts, nil, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err = it.Close(); err != nil {
			srvlog.Error("Could not close filter iterator", log.Ctx{"err": err})
		}
	}()
	for it.Next() {
		if it.Error() != nil {
			return errors.Wrapf(
				err,
				"got iterator error when scanning assertion creations from block %d to %d",
				filterOpts.Start,
				*filterOpts.End,
			)
		}
		_, processErr := retry.UntilSucceeds(ctx, func() (bool, error) {
			return true, s.ProcessAssertionCreation(ctx, protocol.AssertionHash{Hash: it.Event.AssertionHash})
		})
		if processErr != nil {
			return processErr
		}
	}
	return nil
}

func (s *Scanner) ProcessAssertionCreation(
	ctx context.Context,
	assertionHash protocol.AssertionHash,
) error {
	srvlog.Info("Processed assertion creation event", log.Ctx{"validatorName": s.validatorName})
	s.assertionsProcessedCount++
	creationInfo, err := s.chain.ReadAssertionCreationInfo(ctx, assertionHash)
	if err != nil {
		return err
	}
	prevAssertion, err := s.chain.GetAssertion(ctx, protocol.AssertionHash{Hash: creationInfo.ParentAssertionHash})
	if err != nil {
		return err
	}
	hasSecondChild, err := prevAssertion.HasSecondChild()
	if err != nil {
		return err
	}
	if !hasSecondChild {
		srvlog.Info("No fork detected in assertion chain", log.Ctx{"validatorName": s.validatorName})
		return nil
	}
	s.forksDetectedCount++
	execState := protocol.GoExecutionStateFromSolidity(creationInfo.AfterState)
	msgCount, err := s.stateProvider.ExecutionStateMsgCount(ctx, execState)
	switch {
	case errors.Is(err, l2stateprovider.ErrNoExecutionState):
		return nil
	case err != nil:
		return err
	default:
	}

	if s.challengeReader.Mode() == types.DefensiveMode || s.challengeReader.Mode() == types.MakeMode {

		// Generating a random integer between 0 and max delay second to wait before challenging.
		// This is to avoid all validators challenging at the same time.
		mds := 1 // default max delay seconds to 1 to avoid panic
		if s.challengeReader.MaxDelaySeconds() > 1 {
			mds = s.challengeReader.MaxDelaySeconds()
		}
		randSecs, err := randUint64(uint64(mds))
		if err != nil {
			return err
		}
		srvlog.Info("Waiting before challenging", log.Ctx{"delay": randSecs})
		time.Sleep(time.Duration(randSecs) * time.Second)

		if err := s.challengeCreator.ChallengeAssertion(ctx, assertionHash); err != nil {
			return err
		}
		s.challengesSubmittedCount++
		return nil
	}

	srvlog.Error("Detected invalid assertion, but not configured to challenge", log.Ctx{
		"parentAssertionHash":   creationInfo.ParentAssertionHash,
		"detectedAssertionHash": assertionHash,
		"msgCount":              msgCount,
	})
	return nil
}

func (s *Scanner) ForksDetected() uint64 {
	return s.forksDetectedCount
}

func (s *Scanner) ChallengesSubmitted() uint64 {
	return s.challengesSubmittedCount
}

func (s *Scanner) AssertionsProcessed() uint64 {
	return s.assertionsProcessedCount
}

func randUint64(max uint64) (uint64, error) {
	n, err := rand.Int(rand.Reader, new(big.Int).SetUint64(max))
	if err != nil {
		return 0, err
	}
	if !n.IsUint64() {
		return 0, errors.New("not a uint64")
	}
	return n.Uint64(), nil
}
