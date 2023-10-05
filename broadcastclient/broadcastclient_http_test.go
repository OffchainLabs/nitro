package broadcastclient

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/broadcaster"
	"github.com/offchainlabs/nitro/util/signature"
	"github.com/offchainlabs/nitro/wsbroadcastserver"
	"github.com/stretchr/testify/suite"
)

func TestHTTP(t *testing.T) {
	suite.Run(t, new(HTTPSuite))
}

type HTTPSuite struct {
	suite.Suite

	// Broadcaster
	b        *broadcaster.Broadcaster
	bConfig  wsbroadcastserver.BroadcasterConfig
	bErrChan chan error
	chainID  uint64

	// Context
	context context.Context
	cancel  context.CancelFunc

	// Crypto
	privateKey    *ecdsa.PrivateKey
	sequencerAddr *common.Address

	// BroadcastClient
	bc          *BroadcastClient
	bcConfig    Config
	bcErrChan   chan error
	confirmChan chan arbutil.MessageIndex
	ts          *dummyTransactionStreamer
}

func (s *HTTPSuite) SetupTest() {
	// Generate crypto
	privateKey, dataSigner, sequencerAddr, err := generateCrypto()
	Require(s.T(), err)
	s.privateKey = privateKey
	s.sequencerAddr = sequencerAddr

	// Create Broadcaster
	s.bConfig = wsbroadcastserver.DefaultTestBroadcasterConfig
	s.bConfig.HTTP.Enabled = true
	s.chainID = uint64(9742)
	s.bErrChan = make(chan error, 10)
	s.b = broadcaster.NewBroadcaster(
		func() *wsbroadcastserver.BroadcasterConfig { return &s.bConfig },
		s.chainID,
		s.bErrChan,
		dataSigner,
	)

	// Start Broadcaster
	ctx, cancel := context.WithCancel(context.Background())
	s.context = ctx
	s.cancel = cancel
	Require(s.T(), s.b.Initialize())
	Require(s.T(), s.b.Start(s.context))

	// Create BroadcastClient
	s.bcConfig = DefaultTestConfig
	s.bcConfig.HTTP.Port = strconv.Itoa(s.b.HTTPAddr().(*net.TCPAddr).Port)
	s.bcErrChan = make(chan error, 10)
	s.confirmChan = make(chan arbutil.MessageIndex, 10)
	s.ts = NewDummyTransactionStreamer(s.chainID, s.sequencerAddr)
	bc, err := newTestBroadcastClient(
		s.bcConfig,
		s.b.ListenerAddr(),
		s.chainID,
		0,
		s.ts,
		s.confirmChan,
		s.bcErrChan,
		s.sequencerAddr,
	)
	Require(s.T(), err)
	s.bc = bc
}

func (s *HTTPSuite) TearDownTest() {
	// should all tests be doing s.b.StopAndWait()? Maybe I can safely run it twice and it will just exit if already called before in the test
	s.b.StopAndWait()
	s.cancel()
}

// Should we add client & server side compression for HTTP messages too?
func (s *HTTPSuite) TestReceiveMessages() {
	// Send some messages before starting the BroadcastClient (BC) and some
	// messages after. This ensures that some messages are in the HTTP backlog
	// when the BC connects. The BC will take some time to connect on start, it
	// will then receive some of the later messages over WebSocket. This will
	// tell the BC which messages it needs to request over HTTP.
	s.broadcastMessages(0, 5)
	s.bc.Start(s.context)
	defer s.bc.StopAndWait()
	time.Sleep(100 * time.Millisecond) // ensure client connects
	s.broadcastMessages(5, 20)

	// Wait for BroadcastClient to receive all the messages
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	count := 0
	for count < 20 {
		count = s.waitForMessage(timer, count)
	}
}

func (s *HTTPSuite) TestReceiveMessagesMidSequence() {
	// Send some messages before starting the BroadcastClient (BC) and some
	// messages after. This ensures that some messages are in the HTTP backlog
	// when the BC connects. The BC will take some time to connect on start, it
	// will then receive some of the later messages over WebSocket. This will
	// tell the BC which messages it needs to request over HTTP.
	s.broadcastMessages(20, 25)
	s.bc.Start(s.context)
	defer s.bc.StopAndWait()
	time.Sleep(100 * time.Millisecond) // ensure client connects
	s.broadcastMessages(25, 40)

	// Wait for BroadcastClient to receive all the messages
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	count := 0
	for count < 20 {
		count = s.waitForMessage(timer, count)
	}
}

func (s *HTTPSuite) TestInvalidSignature() {
	// Send single message to occupy the HTTP backlog, the client will request these over HTTP
	s.broadcastMessages(0, 5)

	// Create BroadcastClient using wrong certs
	_, _, badSequencerAddr, err := generateCrypto()
	Require(s.T(), err)
	badTS := NewDummyTransactionStreamer(s.chainID, badSequencerAddr)
	badBC, err := newTestBroadcastClient(
		s.bcConfig,
		s.b.ListenerAddr(),
		s.chainID,
		0,
		badTS,
		nil,
		s.bcErrChan,
		badSequencerAddr,
	)
	Require(s.T(), err)
	s.bc = badBC
	s.bc.Start(s.context)
	time.Sleep(100 * time.Millisecond) // ensure client connects

	// The BroadcastClient takes some time to connect, therefore we send around
	// 20 messages. This ensures the client has connected and some messages
	// still come through to the client.
	s.broadcastMessages(5, 20)

	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()
	s.waitForBroadcastClientError(timer, signature.ErrSignatureNotVerified)
}

func (s *HTTPSuite) TestServerClientDisconnect() {
	// Send some messages before starting the BroadcastClient (BC) and some
	// messages after. This ensures that some messages are in the HTTP backlog
	// when the BC connects. The BC will take some time to connect on start, it
	// will then receive some of the later messages over WebSocket. This will
	// tell the BC which messages it needs to request over HTTP.
	s.broadcastMessages(0, 5)
	s.bc.Start(s.context)
	time.Sleep(100 * time.Millisecond) // ensure client connects
	s.broadcastMessages(5, 20)

	// Wait for client to receive all the messages to ensure it is connected
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	count := 0
	for count < 20 {
		count = s.waitForMessage(timer, count)
	}

	// Disconnect the BroadcastClient and wait for the Broadcaster to register
	// that the client has disconnected
	s.bc.StopAndWait()
	disconnectTimer := time.NewTimer(5 * time.Second)
	defer disconnectTimer.Stop()
	for {
		if s.b.ClientCount() == 0 {
			break
		}

		select {
		case err := <-s.bErrChan:
			s.T().Errorf("unexpected Broadcaster error: %v", err)
		case err := <-s.bcErrChan:
			s.T().Errorf("unexpected BroadcastClient error: %v", err)
		case <-disconnectTimer.C:
			s.T().Fatal("timed out waiting for BroadcastClient to disconnect")
		default:
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func (s *HTTPSuite) TestBroadcastConfirmedMessage() {
	// Send some messages before starting the BroadcastClient (BC) and some
	// messages after. This ensures that some messages are in the HTTP backlog
	// when the BC connects. The BC will take some time to connect on start, it
	// will then receive some of the later messages over WebSocket. This will
	// tell the BC which messages it needs to request over HTTP.
	s.broadcastMessages(0, 5)
	s.bc.Start(s.context)
	defer s.bc.StopAndWait()
	time.Sleep(100 * time.Millisecond) // ensure client connects
	s.broadcastMessages(5, 20)

	// Wait for client to receive all the messages to ensure it is connected
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	count := 0
	for count < 20 {
		count = s.waitForMessage(timer, count)
	}

	// Confirm that all messages up to 9 have been seen
	confirm := arbutil.MessageIndex(9)
	s.b.Confirm(confirm)

	// Wait for client to receive confirm message
	confirmTimer := time.NewTimer(5 * time.Second)
	defer confirmTimer.Stop()
	select {
	case err := <-s.bErrChan:
		s.T().Errorf("unexpected Broadcaster error: %v", err)
	case err := <-s.bcErrChan:
		s.T().Errorf("unexpected BroadcastClient error: %v", err)
	case confirmed := <-s.confirmChan:
		if confirmed == confirm {
			s.T().Logf("message %d has been confirmed by BroadcastClient", confirmed)
		} else {
			s.T().Errorf("unexpected message confirmed %d, expected %d", confirmed, confirm)
		}
	case <-confirmTimer.C:
		s.T().Fatal("timed out waiting for BroadcastClient to receive confirmed message")
	}

	// Check that the WebSocket cache count is correct
	n := s.b.GetCachedMessageCount()
	if n != 10 {
		s.T().Errorf("expected cached message count to be 10, count is %d", n)
	}

	// Check that the HTTP backlog count is correct
	// The HTTP backlog stores messages in segments, when messages are deleted
	// only the segments before the confirmed message gets deleted. Tests have
	// a segment size of 3, therefore we only expect 9 messages (3 whole
	// segments) to have been deleted.
	n = s.b.HTTPBacklogMessageCount()
	if n != 11 {
		s.T().Errorf("expected HTTP backlog message count to be 11, count is %d", n)
	}

}

func (s *HTTPSuite) broadcastMessages(start, end int) {
	s.T().Helper()
	for i := start; i < end; i++ {
		Require(s.T(), s.b.BroadcastSingle(arbostypes.TestMessageWithMetadataAndRequestId, arbutil.MessageIndex(i)))
	}
}

func (s *HTTPSuite) waitForMessage(timer *time.Timer, count int) int {
	s.T().Helper()
	return s.waitForResult(timer, count, nil, nil)
}

func (s *HTTPSuite) waitForBroadcastClientError(timer *time.Timer, expErr error) {
	s.T().Helper()
	s.waitForResult(timer, 0, nil, expErr)
}

func (s *HTTPSuite) waitForResult(timer *time.Timer, count int, expBErr, expBCErr error) int {
	s.T().Helper()
	select {
	case err := <-s.bErrChan:
		if expBErr != nil && errors.Is(err, expBErr) {
			s.T().Logf("expected Broadcaster error found: %v", err)
		} else {
			s.T().Errorf("unexpected Broadcaster error: %v", err)
		}
	case err := <-s.bcErrChan:
		if expBCErr != nil && errors.Is(err, expBCErr) {
			s.T().Logf("expected BroadcastClient error found: %v", err)
		} else {
			s.T().Errorf("unexpected BroadcastClient error: %v", err)
		}
	case receivedMsg := <-s.ts.messageReceiver:
		count++
		s.T().Logf("received message: %v", receivedMsg)
	case <-timer.C:
		object := "BroadcastClient"
		waitingFor := "messages"
		if expBErr != nil {
			object = "Broadcaster"
			waitingFor = fmt.Sprintf("an error: %v", expBErr)
		} else if expBCErr != nil {
			waitingFor = fmt.Sprintf("an error: %v", expBCErr)
		}
		s.T().Fatalf("timed out waiting for %s to receive %s", object, waitingFor)
	}
	return count
}

func generateCrypto() (*ecdsa.PrivateKey, signature.DataSignerFunc, *common.Address, error) {
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, nil, nil, err
	}
	dataSigner := signature.DataSignerFromPrivateKey(privateKey)
	sequencerAddr := crypto.PubkeyToAddress(privateKey.PublicKey)
	return privateKey, dataSigner, &sequencerAddr, nil
}
