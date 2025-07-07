package gethexec

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	flag "github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/arbitrum_types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/util/stopwaiter"
)

// Acknowledgement flag that timeboost will wait for
// This is to know sequencer processed Inclusion list succesfully
const ACK_FLAG = 0xc0

var ErrConnectionNotEstablished = errors.New("timeboost txn listener connection not established")

type TimeboostListener struct {
	stopwaiter.StopWaiter
	config         TimeboostListenerConfig
	conn           net.Conn
	connectionLock sync.Mutex
}

type TimeboostListenerConfig struct {
	ListenPort    uint16        `koanf:"listen-port"`
	ReadDeadline  time.Duration `koanf:"read-deadline"`
	WriteDeadline time.Duration `koanf:"write-deadline"`
	MaxBackoff    time.Duration `koanf:"max-backoff"`
}

var DefaultTimeboostListenerConfig = TimeboostListenerConfig{
	ListenPort:    55000,           // Default listen port that timeboost will try and connect to
	ReadDeadline:  4 * time.Second, // Max time we wait on socket `read` to receive inclusion list from timeboost
	WriteDeadline: 4 * time.Second, // Max time we wait while trying to send the acknowledgement back to timeboost
	MaxBackoff:    6 * time.Second, // Max time we wait for backing off and retrying to process the inclusion list when there is no connection
}

func TimeboostListenerConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Uint16(prefix+".listen-port", DefaultTimeboostListenerConfig.ListenPort, "timeboost transaction listener listen port")
	f.Duration(prefix+".read-deadline", DefaultTimeboostListenerConfig.ReadDeadline, "timeboost transaction listener read deadline")
	f.Duration(prefix+".write-deadline", DefaultTimeboostListenerConfig.WriteDeadline, "timeboost transaction listener write deadline")
	f.Duration(prefix+".max-backoff", DefaultTimeboostListenerConfig.MaxBackoff, "timeboost transaction listener max backoff")
}

// Result from the listener accepting new connections
type connectionResult struct {
	conn net.Conn
	err  error
}

func NewTimeboostListener(config TimeboostListenerConfig) (*TimeboostListener, error) {
	return &TimeboostListener{
		config: config,
		conn:   nil,
	}, nil
}

/*
 * This function receives the encoded inclusion list from timeboost and has a deadline for each read operation
 * 1.) Read the encoded inclusion list bytes (u32) size
 * 2.) Read the exact bytes of encoded inclusion list
 */
func (l *TimeboostListener) receiveInclusionList() ([]byte, error) {
	l.connectionLock.Lock()
	defer l.connectionLock.Unlock()
	// Do we have a connection
	if l.conn == nil {
		return nil, ErrConnectionNotEstablished
	}

	// Read encoded inclusion list size (u32)
	deadline := time.Now().Add(l.config.ReadDeadline)
	if err := l.conn.SetReadDeadline(deadline); err != nil {
		return nil, err
	}

	// The size of the inclusion list will be 4 bytes (u32)
	sizeBuf := make([]byte, binary.Size(uint32(0)))
	if _, err := l.conn.Read(sizeBuf); err != nil {
		return nil, err
	}

	// Read inclusion list
	deadline = time.Now().Add(l.config.ReadDeadline)
	if err := l.conn.SetReadDeadline(deadline); err != nil {
		return nil, err
	}

	inclBytes := make([]byte, binary.BigEndian.Uint32(sizeBuf))
	if _, err := l.conn.Read(inclBytes); err != nil {
		return nil, err
	}
	return inclBytes, nil
}

/*
 * This function sends an acknowledgement flag back to timeboost AFTER it successfully processes the transactions
 */
func (l *TimeboostListener) writeAck() error {
	l.connectionLock.Lock()
	defer l.connectionLock.Unlock()
	// Do we have a connection
	if l.conn == nil {
		return ErrConnectionNotEstablished
	}

	// Send back acknowledgement to timeboost so it knows it can move on
	deadline := time.Now().Add(l.config.WriteDeadline)
	if err := l.conn.SetWriteDeadline(deadline); err != nil {
		return err
	}
	if _, err := l.conn.Write([]byte{ACK_FLAG}); err != nil {
		return err
	}
	return nil
}

/*
 * This function closes the connection and reassigns it to nil or a new connection
 * Warning: Be absolutely sure when calling function that you are not be holding the `connectionLock`, this will cause a deadlock
 */
func (l *TimeboostListener) resetConnection(conn net.Conn) {
	l.connectionLock.Lock()
	defer l.connectionLock.Unlock()
	if l.conn != nil {
		if err := l.conn.Close(); err != nil {
			log.Error("timeboost txn listener error closing connection", err)
		}
	}
	// Assign a new connection (this may also be nil which is fine)
	l.conn = conn
}

/*
 * This function listens for incoming connections in its own go routine
 * If there is another successful connection we drop the old connection
 */
func (l *TimeboostListener) connectionHandler(ctx context.Context, port uint16) error {
	addr := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Error("timeboost txn listener failed to start", "port", port, "err", err)
		return err
	}
	defer listener.Close()
	log.Info("timeboost txn listener is listening", "port", port)

	connCh := make(chan connectionResult, 1)
	go func() {
		defer close(connCh)
		// Incase of failures, timeboost will continuously disconnect and reconnect
		// So keep accepting, then send through channel
		for {
			conn, err := listener.Accept()
			connCh <- connectionResult{conn, err}
		}
	}()

	for {
		select {
		case conn := <-connCh:
			if conn.err != nil {
				log.Error("timeboost txn listener connection accept error", "port", port, "err", err)
				continue
			}
			log.Info("received connection", "addr", conn.conn.RemoteAddr())
			// There will only ever be 1 connection at a time between timeboost and sequencer
			// So make sure old connection is closed, and assign it the new connection
			l.resetConnection(conn.conn)
		case <-ctx.Done():
			l.resetConnection(nil)
			log.Info("timeboost txn listener has been terminated")
			return nil
		}
	}
}

/*
 * This function will do 3 steps
 * 1.) Read inclusion list from timeboost
 * 2.) Process the inclusion list in the sequencer
 * 3.) Write an acknowledgement to timeboost notifying it succeeded
 * If there are any failures, it will reset the connection and wait for timeboost to reconnect and resend
 */
func process(
	ctx context.Context,
	l *TimeboostListener,
	backoff *time.Duration,
	processInclusionListFunc func(context.Context, []byte, *arbitrum_types.ConditionalOptions) error,
) time.Duration {
	maxBackoff := l.config.MaxBackoff
	currentBackoff := *backoff

	// Get inclusion list bytes
	inclBytes, err := l.receiveInclusionList()
	if err != nil {
		l.resetConnection(nil)
		log.Warn("error receiving inclusion list", "err", err, "backoff", currentBackoff)
		// only do exponential delay if waiting for connection
		if errors.Is(err, ErrConnectionNotEstablished) {
			*backoff = min(currentBackoff*2, maxBackoff)
			return currentBackoff
		}
		*backoff = time.Second
		return time.Second
	}

	*backoff = time.Second
	currentBackoff = *backoff

	// Decode and process inclusion list
	if err := processInclusionListFunc(ctx, inclBytes, nil); err != nil {
		l.resetConnection(nil)
		log.Warn("error processing inclusion list", "err", err, "backoff", currentBackoff)
		return currentBackoff
	}

	// Send acknowledgement to timeboost we received and processed
	if err := l.writeAck(); err != nil {
		l.resetConnection(nil)
		log.Warn("error writing ack to timeboost", "err", err, "backoff", currentBackoff)
		// only do exponential delay if waiting for connection
		if errors.Is(err, ErrConnectionNotEstablished) {
			*backoff = min(currentBackoff*2, maxBackoff)
			return currentBackoff
		}
		return currentBackoff
	}

	return 0
}

func (l *TimeboostListener) Start(
	ctx context.Context,
	processInclusionListFunc func(context.Context, []byte, *arbitrum_types.ConditionalOptions) error,
) error {
	if l.config.MaxBackoff > 10*time.Second || l.config.MaxBackoff < 5*time.Second {
		panic("max backoff needs to be between 5 and 10 seconds")
	}
	if l.config.ReadDeadline > 10*time.Second || l.config.ReadDeadline < 3*time.Second {
		panic("read deadline needs to be between 3 and 10 seconds")
	}
	if l.config.WriteDeadline > 10*time.Second || l.config.WriteDeadline < 3*time.Second {
		panic("write deadline needs to be between 3 and 10 seconds")
	}

	l.StopWaiter.Start(ctx, l)

	// Connection handler thread
	l.LaunchThread(func(ctx context.Context) {
		err := l.connectionHandler(ctx, l.config.ListenPort)
		if err != nil {
			panic(err)
		}
	})

	// Process inclusion list thread
	backoff := time.Second
	if err := l.CallIterativelySafe(func(ctx context.Context) time.Duration {
		return process(ctx, l, &backoff, processInclusionListFunc)
	}); err != nil {
		log.Error("timeboost txn listener failed to start inclusion list processor")
		return err
	}
	return nil
}

func (l *TimeboostListener) StopAndWait() {
	l.StopWaiter.StopAndWait()
}
