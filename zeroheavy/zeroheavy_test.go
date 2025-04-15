// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

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
	"github.com/offchainlabs/nitro/util/colors"
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

func l1Cost(data []byte) int {
	cost := 4 * len(data)
	for _, b := range data {
		if b != 0 {
			cost += 12
		}
	}
	return cost
}

func TestZeroHeavyRandomDataRandom(t *testing.T) {
	rand.Seed(time.Now().UTC().UnixNano())

	trials := 1024
	avg := 0.0
	best := 0.0
	worst := 100.0

	for i := 0; i < trials; i++ {
		size := 1 + rand.Uint64()%4096
		inBuf := testhelpers.RandomizeSlice(make([]byte, size))
		enc := NewZeroheavyEncoder(bytes.NewReader(inBuf))
		encoded, err := io.ReadAll(enc)
		ShowError(t, err)

		improvement := 100.0 * float64(l1Cost(inBuf)-l1Cost(encoded)) / float64(l1Cost(inBuf))
		if improvement > best {
			best = improvement
			colors.PrintGrey("best  ", len(encoded), "/", size, "\t", l1Cost(encoded), "/", l1Cost(inBuf))
		}
		if improvement < worst {
			worst = improvement
			colors.PrintGrey("worst ", len(encoded), "/", size, "\t", l1Cost(encoded), "/", l1Cost(inBuf))
		}

		avg += improvement / float64(trials)

		dec := NewZeroheavyDecoder(bytes.NewReader(encoded))
		res, err := io.ReadAll(dec)
		ShowError(t, err)
		if !bytes.Equal(inBuf, res) {
			Fail(t, size, inBuf)
		}
	}

	colors.PrintBlue("avg   improvement ", avg)
	colors.PrintBlue("best  improvement ", best)
	colors.PrintBlue("worst improvement ", worst)
}

func TestZeroHeavyRandomDataBrotli(t *testing.T) {
	rand.Seed(time.Now().UTC().UnixNano())

	trials := 256
	avg := 0.0
	best := 0.0
	worst := 100.0

	for i := 0; i < trials; i++ {

		// Compress a low-entropy input
		size := 100 + rand.Uint64()%2048
		randomBytes := testhelpers.RandomizeSlice(make([]byte, size))
		for i := range randomBytes {
			randomBytes[i] /= 8
		}
		for i, b := range randomBytes {
			if b < 0x14 {
				randomBytes[i] = 0
			}
		}
		input, err := arbcompress.CompressWell(randomBytes)
		Require(t, err)

		ShowError(t, err)
		enc := NewZeroheavyEncoder(bytes.NewReader(input))
		encoded, err := io.ReadAll(enc)
		ShowError(t, err)

		improvement := 100.0 * float64(l1Cost(input)-l1Cost(encoded)) / float64(l1Cost(input))
		if improvement > best {
			best = improvement
			colors.PrintGrey("best  ", len(encoded), "/", size, "\t", l1Cost(encoded), "/", l1Cost(input))
		}
		if improvement < worst {
			worst = improvement
			colors.PrintGrey("worst ", len(encoded), "/", size, "\t", l1Cost(encoded), "/", l1Cost(input))
		}

		avg += improvement / float64(trials)

		dec := NewZeroheavyDecoder(bytes.NewReader(encoded))
		res, err := io.ReadAll(dec)
		ShowError(t, err)
		if !bytes.Equal(input, res) {
			Fail(t, size, input)
		}
	}

	colors.PrintBlue("avg   improvement ", avg)
	colors.PrintBlue("best  improvement ", best)
	colors.PrintBlue("worst improvement ", worst)
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
