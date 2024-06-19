package assertions

import (
	"context"
	"fmt"
	"strings"
	"time"

	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	solimpl "github.com/OffchainLabs/bold/chain-abstraction/sol-implementation"
	"github.com/OffchainLabs/bold/challenge-manager/types"
	"github.com/OffchainLabs/bold/containers/option"
	retry "github.com/OffchainLabs/bold/runtime"
	"github.com/ethereum/go-ethereum/log"
)

func (m *Manager) queueCanonicalAssertionsForConfirmation(ctx context.Context) {
	for {
		select {
		case canonical := <-m.observedCanonicalAssertions:
			m.LaunchThread(func(ctx context.Context) { m.keepTryingAssertionConfirmation(ctx, canonical) })
		case <-ctx.Done():
			return
		}
	}
}

func (m *Manager) keepTryingAssertionConfirmation(ctx context.Context, assertionHash protocol.AssertionHash) {
	// Only resolve mode strategies or higher should be confirming assertions.
	if m.challengeReader.Mode() < types.ResolveMode {
		return
	}
	creationInfo, err := retry.UntilSucceeds(ctx, func() (*protocol.AssertionCreatedInfo, error) {
		return m.chain.ReadAssertionCreationInfo(ctx, assertionHash)
	})
	if err != nil {
		log.Error("Could not get assertion creation info", "err", err)
		return
	}
	prevCreationInfo, err := retry.UntilSucceeds(ctx, func() (*protocol.AssertionCreatedInfo, error) {
		return m.chain.ReadAssertionCreationInfo(ctx, protocol.AssertionHash{Hash: creationInfo.ParentAssertionHash})
	})
	if err != nil {
		log.Error("Could not get prev assertion creation info", "err", err)
		return
	}
	ticker := time.NewTicker(m.confirmationAttemptInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			parentAssertion, err := m.chain.GetAssertion(ctx, protocol.AssertionHash{Hash: creationInfo.ParentAssertionHash})
			if err != nil {
				log.Error("Could not get parent assertion", "err", err)
				continue
			}
			parentAssertionHasSecondChild, err := parentAssertion.HasSecondChild()
			if err != nil {
				log.Error("Could not confirm if parent assertion has second child", "err", err)
				continue
			}
			// Assertions that have a rival assertion cannot be confirmed by time.
			if parentAssertionHasSecondChild {
				return
			}
			confirmed, err := solimpl.TryConfirmingAssertion(ctx, creationInfo.AssertionHash, prevCreationInfo.ConfirmPeriodBlocks+creationInfo.CreationBlock, m.chain, m.averageTimeForBlockCreation, option.None[protocol.EdgeId]())
			if err != nil {
				if !strings.Contains(err.Error(), "PREV_NOT_LATEST_CONFIRMED") {
					log.Error("Could not confirm assertion", "err", err, "assertionHash", assertionHash.Hash)
					errorConfirmingAssertionByTimeCounter.Inc(1)
				}
				continue
			}
			if confirmed {
				assertionConfirmedCounter.Inc(1)
				log.Info("Confirmed assertion by time", "assertionHash", creationInfo.AssertionHash)
				return
			}
		}
	}
}

func (m *Manager) updateLatestConfirmedMetrics(ctx context.Context) {
	ticker := time.NewTicker(m.confirmationAttemptInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			latestConfirmed, err := m.chain.LatestConfirmed(ctx)
			if err != nil {
				log.Debug("Could not fetch latest confirmed assertion", "err", err)
				continue
			}
			info, err := m.chain.ReadAssertionCreationInfo(ctx, latestConfirmed.Id())
			if err != nil {
				log.Debug("Could not fetch latest confirmed assertion", "err", err)
				continue
			}
			afterState := protocol.GoExecutionStateFromSolidity(info.AfterState)
			log.Info("Latest confirmed assertion", "assertionAfterState", fmt.Sprintf("%+v", afterState))

			// TODO: Check if the latest assertion that was confirmed is one we agree with.
			latestConfirmedAssertionGauge.Update(int64(latestConfirmed.CreatedAtBlock()))
		case <-ctx.Done():
			return
		}
	}
}
