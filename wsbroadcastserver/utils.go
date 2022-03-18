package wsbroadcastserver

import (
	"context"
	"errors"
	"io"
	"io/ioutil"
	"net"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
)

type chainedReader struct {
	readers []io.Reader
}

func (cr *chainedReader) Read(b []byte) (n int, err error) {
	for _, r := range cr.readers {
		n, err = r.Read(b)
		if errors.Is(err, io.EOF) || n == 0 {
			continue
		}
		break
	}
	return
}

func (cr *chainedReader) add(r io.Reader) *chainedReader {
	if r != nil {
		cr.readers = append(cr.readers, r)
	}
	return cr
}

func ReadData(ctx context.Context, conn net.Conn, earlyFrameData io.Reader, idleTimeout time.Duration, state ws.State) ([]byte, ws.OpCode, error) {

	controlHandler := wsutil.ControlFrameHandler(conn, state)
	reader := wsutil.Reader{
		Source:          (&chainedReader{}).add(earlyFrameData).add(conn),
		State:           state,
		CheckUTF8:       true,
		SkipHeaderCheck: false,
		OnIntermediate:  controlHandler,
	}

	// Remove timeout when leaving this function
	defer func(conn net.Conn) {
		err := conn.SetReadDeadline(time.Time{})
		if err != nil && !strings.Contains(err.Error(), "use of closed network connection") {
			log.Error("error removing read deadline", "err", err)
		}
	}(conn)

	for {
		select {
		case <-ctx.Done():
			return nil, 0, nil
		default:
		}

		err := conn.SetReadDeadline(time.Now().Add(idleTimeout))
		if err != nil {
			return nil, 0, err
		}

		// Control packet may be returned even if err set
		header, err := reader.NextFrame()
		if header.OpCode.IsControl() {
			// Control packet may be returned even if err set
			if err2 := controlHandler(header, &reader); err2 != nil {
				return nil, 0, err2
			}

			// Discard any data after control packet
			if err2 := reader.Discard(); err2 != nil {
				return nil, 0, err2
			}

			return nil, 0, nil
		}
		if err != nil {
			return nil, 0, err
		}

		if header.OpCode != ws.OpText &&
			header.OpCode != ws.OpBinary {
			if err := reader.Discard(); err != nil {
				return nil, 0, err
			}
			continue
		}

		data, err := ioutil.ReadAll(&reader)

		return data, header.OpCode, err
	}
}
