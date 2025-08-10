package arbnode

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/espressotee"
	"github.com/offchainlabs/nitro/solgen/go/espressogen"
)

// Reads state from external sources and resets the espresso streamer to start producing
// messages from hotshot based on the source of truth on the parent chain
func (b *BatchPoster) resetStreamerToParentChainOrConfigHotshotBlock(messageCount arbutil.MessageIndex, ctx context.Context) {
	hotshotBlock := b.fetchHotshotBlockFromLastCheckpoint(ctx)
	if hotshotBlock == 0 {
		// if there hasn't been a batch posted, or we encountered an error, start reading from the configured hotshot block number.
		hotshotBlock = b.config().HotShotBlock
	}
	b.espressoStreamer.Reset(uint64(messageCount), hotshotBlock)
}

// fetchHotshotBlockFromLastCheckpoint:
// This function uses the sequencer inbox bridgegen contract to filter for logs related to the TEESignatureVerified events
// If any of these events are encountered, it checks the log iterator for the data about this event.
// Return:
// returns the Hotshot height of the last event in the iterator returned from FilterTEESignatureVerified()
// representing the most recently emitted hotshotblock height. Any errors encountered will result in 0 being returned.
func (b *BatchPoster) fetchHotshotBlockFromLastCheckpoint(ctx context.Context) uint64 {
	pollingStep := b.config().EspressoEventPollingStep
	header, err := b.l1Reader.LastHeader(ctx)
	if err != nil {
		log.Error("Failed to fetch last header from parent chain", "err", err)
		return 0
	}

	var lastHotshotHeight uint64 = 0
	// Prevent unsigned integer underflow: in Go, subtracting a larger value
	// from a smaller uint64 will wrap around to a very large number.
	for i := header.Number.Uint64(); i >= b.config().HotShotFirstPostingBlock; i -= min(i, pollingStep) {
		start := i - min(i, pollingStep)
		if start < b.config().HotShotFirstPostingBlock {
			start = b.config().HotShotFirstPostingBlock
		}
		filterOpts := bind.FilterOpts{
			Start:   start,
			End:     &i,
			Context: ctx,
		}

		logIterator, err := b.seqInbox.FilterTEESignatureVerified(&filterOpts, []*big.Int{}, []*big.Int{})
		if err != nil {
			log.Error("Failed to obtain iterator for logs for block", "blockNumber", i, "err", err)
			continue
		}

		if logIterator == nil {
			continue
		}

		for logIterator.Next() {
			lastHotshotHeight = logIterator.Event.HotshotHeight.Uint64()
		}

		if lastHotshotHeight > 0 {
			return lastHotshotHeight
		}
	}

	log.Warn("No logs found for Hotshot block")
	return 0
}

func setupNitroVerifier(teeVerifier *espressogen.IEspressoTEEVerifier, l1Client *ethclient.Client) (espressotee.EspressoNitroTEEVerifierInterface, error) {
	// Setup nitro contract interface
	nitroAddr, err := teeVerifier.EspressoNitroTEEVerifier(&bind.CallOpts{})
	if err != nil {
		return nil, fmt.Errorf("failed to get nitro tee verifier address from caller: %w", err)
	}
	log.Info("successfully retrieved nitro contract verifier address", "address", nitroAddr)

	nitroVerifierBindings, err := espressogen.NewIEspressoNitroTEEVerifier(
		nitroAddr,
		l1Client)
	if err != nil {
		return nil, err
	}
	nitroVerifier := espressotee.NewEspressoNitroTEEVerifier(nitroVerifierBindings, l1Client, nitroAddr)
	return nitroVerifier, nil
}
