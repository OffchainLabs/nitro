package validator

import (
	"context"
	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	"github.com/OffchainLabs/challenge-protocol-v2/state-manager"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/common"
	"github.com/sirupsen/logrus"
	"time"
)

type edgeTrackerConfig struct {
	actEveryNSeconds      time.Duration
	timeRef               util.TimeReference
	challengePeriodLength time.Duration
	chain                 protocol.Protocol
	stateManager          statemanager.Manager
	validatorName         string
	validatorAddress      common.Address
}

type edgeTracker struct {
	cfg       *edgeTrackerConfig
	challenge protocol.SpecChallenge
	edge      protocol.SpecEdge
	fsm       *util.Fsm[edgeTrackerAction, edgeTrackerState]
}

func newEdgeTracker(
	cfg *edgeTrackerConfig,
	challenge protocol.SpecChallenge,
	edge protocol.SpecEdge,
	fsmOpts ...util.FsmOpt[edgeTrackerAction, edgeTrackerState],
) (*edgeTracker, error) {
	fsm, err := newEdgeTrackerFsm(
		edgeStarted,
		util.WithTrackedTransitions[edgeTrackerAction, edgeTrackerState](),
	)
	if err != nil {
		return nil, err
	}
	return &edgeTracker{
		cfg:       cfg,
		challenge: challenge,
		edge:      edge,
		fsm:       fsm,
	}, nil
}

func (et *edgeTracker) spawn(ctx context.Context) {
	startHeight, startCommit := et.edge.StartCommitment()
	endHeight, endCommit := et.edge.TargetCommitment()
	fields := logrus.Fields{
		"start":         startHeight,
		"end":           endHeight,
		"startMerkle":   util.Trunc(startCommit.Bytes()),
		"endMerkle":     util.Trunc(endCommit.Bytes()),
		"validatorName": et.cfg.validatorName,
		"challengeType": et.challenge.GetType(),
		"address":       util.Trunc(et.cfg.validatorAddress.Bytes()),
	}
	log.WithFields(fields).Info("Tracking edge vertex")

	t := et.cfg.timeRef.NewTicker(et.cfg.actEveryNSeconds)
	defer t.Stop()
	for {
		select {
		case <-t.C():
			// Check if the associated edge or challenge are confirmed,
			// or if a rival edge exists that has been confirmed before acting.
			shouldComplete, err := et.shouldComplete(ctx)
			if err != nil {
				log.WithError(err).WithFields(fields).Error("Could not check if edge tracker should complete")
				continue
			}
			if shouldComplete {
				log.WithFields(fields).Debug("Edge tracker received notice of a confirmation, exiting")
				return
			}
			if err := et.act(ctx); err != nil {
				log.Error(err)
			}
		case <-ctx.Done():
			log.WithFields(fields).Debug("Edge tracker goroutine exiting")
			return
		}
	}
}

func (et *edgeTracker) shouldComplete(ctx context.Context) (bool, error) {
	return false, nil
}

func (et *edgeTracker) act(ctx context.Context) error {
	return nil
}
