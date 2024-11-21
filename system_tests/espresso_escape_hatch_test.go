package arbtest

import (
	"context"
	"encoding/json"
	"os/exec"
	"testing"
	"time"

	lightclientmock "github.com/EspressoSystems/espresso-sequencer-go/light-client-mock"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbutil"
)

func TestEspressoEscapeHatch(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Disabling the delayed sequencer helps up check the
	// message count easily
	builder, cleanup := createL1AndL2Node(ctx, t, false)
	defer cleanup()

	err := waitForL1Node(ctx)
	Require(t, err)

	cleanEspresso := runEspresso()
	defer cleanEspresso()

	// wait for the builder
	err = waitForEspressoNode(ctx)
	Require(t, err)

	l2Node := builder.L2
	l2Info := builder.L2Info

	// wait for the latest hotshot block
	err = waitFor(ctx, func() bool {
		out, err := exec.Command("curl", "http://127.0.0.1:41000/status/block-height", "-L").Output()
		if err != nil {
			return false
		}
		h := 0
		err = json.Unmarshal(out, &h)
		if err != nil {
			return false
		}
		// Wait for the hotshot to generate some blocks to better simulate the real-world environment.
		// Chosen based on intuition; no empirical data supports this value.
		return h > 10
	})
	Require(t, err)

	address := common.HexToAddress(lightClientAddress)
	txOpts := builder.L1Info.GetDefaultTransactOpts("Faucet", ctx)

	if builder.L2.ConsensusNode.TxStreamer.UseEscapeHatch {
		t.Fatal("testing not using escape hatch first")
	}
	log.Info("Checking turning off the escape hatch")

	// Start to check the escape hatch

	// Freeze the l1 height
	err = lightclientmock.FreezeL1Height(t, builder.L1.Client, address, &txOpts)
	Require(t, err)
	log.Info("waiting for light client to report hotshot is down")
	err = waitForWith(ctx, 10*time.Minute, 10*time.Second, func() bool {
		log.Info("waiting for hotshot down")
		return builder.L2.ConsensusNode.TxStreamer.HotshotDown
	})
	Require(t, err)

	log.Info("light client has reported that hotshot is down")

	// Wait for the switch to be totally finished
	currMsg, err := builder.L2.ConsensusNode.TxStreamer.GetMessageCount()
	Require(t, err)

	log.Info("waiting for message count", "currMsg", currMsg)
	var validatedMsg arbutil.MessageIndex
	err = waitForWith(ctx, 6*time.Minute, 60*time.Second, func() bool {
		validatedCnt := builder.L2.ConsensusNode.BlockValidator.Validated(t)
		log.Info("Validation status", "validatedCnt", validatedCnt, "currCnt", currMsg)
		if validatedCnt >= currMsg {
			validatedMsg = validatedCnt
			return true
		}
		return false
	})
	Require(t, err)
	err = checkTransferTxOnL2(t, ctx, l2Node, "User12", l2Info)
	Require(t, err)
	err = checkTransferTxOnL2(t, ctx, l2Node, "User13", l2Info)
	Require(t, err)

	time.Sleep(20 * time.Second)
	validated := builder.L2.ConsensusNode.BlockValidator.Validated(t)
	if validated > validatedMsg {
		t.Fatal("Escape hatch is not used. Validated messages should not increase anymore")
	}

	log.Info("setting hotshot back")
	// Unfreeze the l1 height
	err = lightclientmock.UnfreezeL1Height(t, builder.L1.Client, address, &txOpts)
	Require(t, err)

	// Check if the validated count is increasing after hotshot goes back live
	err = waitForWith(ctx, 3*time.Minute, 20*time.Second, func() bool {
		validated := builder.L2.ConsensusNode.BlockValidator.Validated(t)
		return validated > validatedMsg
	})
	Require(t, err)

	log.Info("testing escape hatch")
	// Modify it manually
	builder.L2.ConsensusNode.TxStreamer.UseEscapeHatch = true

	err = lightclientmock.FreezeL1Height(t, builder.L1.Client, address, &txOpts)
	Require(t, err)

	err = waitForWith(ctx, 10*time.Minute, 10*time.Second, func() bool {
		log.Info("waiting for hotshot down")
		return builder.L2.ConsensusNode.TxStreamer.HotshotDown
	})
	Require(t, err)

	err = checkTransferTxOnL2(t, ctx, l2Node, "User14", l2Info)
	Require(t, err)
	err = checkTransferTxOnL2(t, ctx, l2Node, "User15", l2Info)
	Require(t, err)
	currMsg, err = builder.L2.ConsensusNode.TxStreamer.GetMessageCount()
	Require(t, err)
	// Escape hatch is on, so the validated count should keep increasing
	err = waitForWith(ctx, 10*time.Minute, 10*time.Second, func() bool {
		validated := builder.L2.ConsensusNode.BlockValidator.Validated(t)
		return validated >= currMsg
	})
	Require(t, err)
	// TODO: Find a way to check if any hotshot transaction is submitted,
	// then set the hotshot live again.
}
