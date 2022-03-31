// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package zeroheavy

import (
	"bytes"
	"errors"
	"io"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/offchainlabs/nitro/arbcompress"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func TestZeroheavyNullInput(t *testing.T) {
	inBuf := []byte{}
	source := bytes.NewReader(inBuf)
	enc := NewZeroheavyEncoder(source)
	dec := NewZeroheavyDecoder(enc)

	var buf [256]byte
	n, err := dec.Read(buf[:])
	if !errors.Is(err, io.EOF) {
		Fail(t)
	}
	if n != 0 {
		Fail(t, n, buf[0])
	}
}

func TestZeroHeavyOneByte(t *testing.T) {
	for i := 0; i < 256; i++ {
		inBuf := []byte{byte(i)}
		source := bytes.NewReader(inBuf)
		enc := NewZeroheavyEncoder(source)
		dec := NewZeroheavyDecoder(enc)

		buf, err := io.ReadAll(dec)
		ShowError(t, err)

		if len(buf) != 1 {
			Fail(t, i, len(buf))
		}
		if buf[0] != byte(i) {
			Fail(t, buf[0], i)
		}
	}
}

func TestZeroHeavyRandomData(t *testing.T) {
	rand.Seed(time.Now().UTC().UnixNano())
	for i := 0; i < 1024; i++ {
		size := rand.Uint64() % 4096
		inBuf := testhelpers.RandomizeSlice(make([]byte, size))
		dec := NewZeroheavyDecoder(NewZeroheavyEncoder(bytes.NewReader(inBuf)))
		res, err := io.ReadAll(dec)
		ShowError(t, err)
		if !bytes.Equal(inBuf, res) {
			Fail(t, size, inBuf)
		}
	}
}

func TestZeroHeavyAndBrotli(t *testing.T) {
	inData, err := os.ReadFile("../go.sum")
	ShowError(t, err)

	bout, err := arbcompress.CompressWell(inData)
	ShowError(t, err)

	zhout, err := io.ReadAll(NewZeroheavyDecoder(NewZeroheavyEncoder(bytes.NewReader(bout))))
	ShowError(t, err)

	res, err := arbcompress.Decompress(zhout, len(inData))
	ShowError(t, err)

	if !bytes.Equal(inData, res) {
		Fail(t)
	}
}
