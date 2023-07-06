// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/challenge-protocol-v2/blob/main/LICENSE

// Package assertions contains testing utilities for posting and scanning for
// assertions on chain, which are useful for simulating the responsibilities
// of Arbitrum Nitro and initiating challenges as needed using our challenge manager.
package assertions

import (
	"context"
	"time"

	protocol "github.com/OffchainLabs/challenge-protocol-v2/chain-abstraction"
	"github.com/OffchainLabs/challenge-protocol-v2/challenge-manager/types"
	l2stateprovider "github.com/OffchainLabs/challenge-protocol-v2/layer2-state-provider"
	retry "github.com/OffchainLabs/challenge-protocol-v2/runtime"
	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/rollupgen"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "assertion-scanner")

// Scanner checks for posted, onchain assertions via a polling mechanism since the latest confirmed,
// up to the latest block, and keeps doing so as the chain advances. With each observed assertion,
// it determines whether or not it should challenge it.
type Scanner struct {
	chain               protocol.AssertionChain
	backend             bind.ContractBackend
	challengeCreator    types.ChallengeCreator
	challengeModeReader types.ChallengeModeReader
	stateProvider       l2stateprovider.Provider
	pollInterval        time.Duration
	rollupAddr          common.Address
	validatorName       string
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
		chain:               chain,
		backend:             backend,
		stateProvider:       stateProvider,
		challengeCreator:    challengeManager,
		challengeModeReader: challengeManager,
		rollupAddr:          rollupAddr,
		validatorName:       validatorName,
		pollInterval:        pollInterval,
	}
}

// Scan the blockchain for assertion creation events in a polling manner
// from the latest confirmed assertion.
func (s *Scanner) Start(ctx context.Context) {
	latestConfirmed, err := s.chain.LatestConfirmed(ctx)
	if err != nil {
		log.Error(err)
		return
	}
	fromBlock, err := latestConfirmed.CreatedAtBlock()
	if err != nil {
		log.Error(err)
		return
	}

	filterer, err := retry.UntilSucceeds(ctx, func() (*rollupgen.RollupUserLogicFilterer, error) {
		return rollupgen.NewRollupUserLogicFilterer(s.rollupAddr, s.backend)
	})
	if err != nil {
		log.Error(err)
		return
	}
	ticker := time.NewTicker(s.pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			latestBlock, err := s.backend.HeaderByNumber(ctx, nil)
			if err != nil {
				log.Error(err)
				continue
			}
			if !latestBlock.Number.IsUint64() {
				log.Fatal("Latest block number was not a uint64")
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
				log.Error(err)
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
			log.WithError(err).Error("Could not close filter iterator")
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
			return true, s.ProcessAssertionCreation(ctx, it.Event.AssertionHash)
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
	log.WithFields(logrus.Fields{
		"validatorName": s.validatorName,
	}).Info("Processed assertion creation event")
	creationInfo, err := s.chain.ReadAssertionCreationInfo(ctx, assertionHash)
	if err != nil {
		return err
	}
	prevAssertion, err := s.chain.GetAssertion(ctx, protocol.AssertionHash(creationInfo.ParentAssertionHash))
	if err != nil {
		return err
	}
	hasSecondChild, err := prevAssertion.HasSecondChild()
	if err != nil {
		return err
	}
	if !hasSecondChild {
		log.WithFields(logrus.Fields{
			"validatorName": s.validatorName,
		}).Info("No fork detected in assertion chain")
		return nil
	}
	execState := protocol.GoExecutionStateFromSolidity(creationInfo.AfterState)
	msgCount, agreesWithAssertion, err := s.stateProvider.ExecutionStateMsgCount(ctx, execState)
	if err != nil {
		return err
	}
	if !agreesWithAssertion {
		return nil
	}

	if s.challengeModeReader.Mode() == types.DefensiveMode || s.challengeModeReader.Mode() == types.MakeMode {
		if err := s.challengeCreator.ChallengeAssertion(ctx, assertionHash); err != nil {
			return err
		}
		return nil
	}

	log.WithFields(logrus.Fields{
		"parentAssertionHash":   creationInfo.ParentAssertionHash,
		"detectedAssertionHash": assertionHash,
		"msgCount":              msgCount,
	}).Error("Detected invalid assertion, but not configured to challenge")
	return nil
}
