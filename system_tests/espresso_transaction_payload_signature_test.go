package arbtest

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"testing"
	"time"

	lightclient "github.com/EspressoSystems/espresso-sequencer-go/light-client"
	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
)

func TestEspressoTransactionSignatureForSovereignSequencer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	valNodeCleanup := createValidationNode(ctx, t, true)
	defer valNodeCleanup()

	builder, cleanup := createL1AndL2Node(ctx, t)
	defer cleanup()

	l2Node := builder.L2
	l2Info := builder.L2Info
	l1Info := builder.L1Info

	err := waitForL1Node(ctx)
	Require(t, err)

	cleanEspresso := runEspresso()
	defer cleanEspresso()

	// wait for the builder
	err = waitForEspressoNode(ctx)
	Require(t, err)

	lightClientReader, err := lightclient.NewLightClientReader(common.HexToAddress(lightClientAddress), builder.L1.Client)
	Require(t, err)
	// wait for hotshot liveness

	err = waitForHotShotLiveness(ctx, lightClientReader)
	Require(t, err)
	msgCnt, err := l2Node.ConsensusNode.TxStreamer.GetMessageCount()
	Require(t, err)
	flag := true

	// Wait for the msgCnt to stop increasing
	err = waitForWith(ctx, 3*time.Minute, 60*time.Second, func() bool {
		if flag {
			flag = false
			return false
		}
		cnt, err := l2Node.ConsensusNode.TxStreamer.GetMessageCount()
		if err != nil {
			return false
		}
		if cnt == msgCnt {
			return true
		}

		msgCnt = cnt
		return false

	})
	Require(t, err)

	err = checkTransferTxOnL2(t, ctx, l2Node, "User14", l2Info)
	Require(t, err)

	privateKey := l1Info.GetInfoWithPrivKey("Sequencer").PrivateKey

	message, err := l2Node.ConsensusNode.TxStreamer.GetMessage(msgCnt)
	Require(t, err)

	err = checkSignatureValidation(message, privateKey.PublicKey)
	Require(t, err)

}

func checkSignatureValidation(message *arbostypes.MessageWithMetadata, publicKey ecdsa.PublicKey) error {
	txns, _, err := arbos.ParseEspressoMsg(message.Message)
	if err != nil {
		return err
	}

	if len(txns) < 1 || len(txns[0]) < 65 {
		return fmt.Errorf("txns length is %d should be at least 1 and should contain the payload signature", len(txns))
	}

	// signature will always be 65 bytes
	payloadSignature := txns[0][:65]

	txnsHash := crypto.Keccak256(txns[0][65:])

	publicKeyBytes := crypto.FromECDSAPub(&publicKey)

	if !crypto.VerifySignature(publicKeyBytes, txnsHash, payloadSignature) {
		return err
	}
	return nil
}
