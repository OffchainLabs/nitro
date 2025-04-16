// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package zeroheavy

import (
	"errors"
	"io"
)

type ZeroheavyEncoder struct {
	inner              io.Reader
	buffer             byte
	bitsReadFromBuffer uint8
	nowInPadding       bool
	atEof              bool
}

func NewZeroheavyEncoder(inner io.Reader) *ZeroheavyEncoder {
	return &ZeroheavyEncoder{inner, 0, 8, false, false}
}

func (enc *ZeroheavyEncoder) nextInputBit() (bool, error) {
	if enc.nowInPadding {
		return true, nil
	}
	if enc.bitsReadFromBuffer == 8 {
		var buf [1]byte
		_, err := enc.inner.Read(buf[:])
		if errors.Is(err, io.EOF) {
			// we're in padding mode now; we'll emit a false, then as many trues as needed
			enc.nowInPadding = true
			return false, nil
		}
		if err != nil {
			return false, err
		}
		enc.bitsReadFromBuffer = 0
		enc.buffer = buf[0]
	}
	ret := (enc.buffer & (1 << (7 - enc.bitsReadFromBuffer))) != 0
	enc.bitsReadFromBuffer++
	return ret, nil
}

func (enc *ZeroheavyEncoder) readOne() (byte, error) {
	if enc.atEof {
		return 0, io.EOF
	}
	b, err := enc.readOneImpl()
	if err != nil {
		return b, err
	}
	if enc.nowInPadding {
		// our input is at EOF, and we have consumed some padding, so this should be the last byte produced
		enc.atEof = true
	}
	return b, nil
}

func (enc *ZeroheavyEncoder) readOneImpl() (byte, error) {
	firstBit, err := enc.nextInputBit()
	if err != nil {
		return 0, err
	}
	if !firstBit {
		secondBit, err := enc.nextInputBit()
		if err != nil {
			return 0, err
		}
		if !secondBit {
			return 0, nil
		}
		ret := byte(1)
		for i := 0; i < 6; i++ {
			nextBit, err := enc.nextInputBit()
			if err != nil {
				return 0, err
			}
			ret <<= 1
			if nextBit {
				ret++
			}
		}
		if ret == 64 {
			return 1, nil
		}
		ret = (ret << 1) & 0x7f
		nextBit, err := enc.nextInputBit()
		if err != nil {
			return 0, err
		}
		if nextBit {
			ret++
		}
		return ret, nil
	}
	ret := byte(1) // first bit is 1
	for i := 0; i < 7; i++ {
		ret <<= 1
		nextBit, err := enc.nextInputBit()
		if err != nil {
			return 0, err
		}
		if nextBit {
			ret += 1
		}
	}
	return ret, nil
}

func (enc *ZeroheavyEncoder) Read(p []byte) (int, error) {
	for i := range p {
		b, err := enc.readOne()
		if err != nil {
			return i, err
		}
		p[i] = b
	}
	return len(p), nil
}

type ZeroheavyDecoder struct {
	source          io.Reader
	buffer          []bool
	eofAfterBuffer  bool
	deferredZero    bool
	numDeferredOnes uint
}

func NewZeroheavyDecoder(source io.Reader) *ZeroheavyDecoder {
	return &ZeroheavyDecoder{source, []bool{}, false, false, 0}
}

func (br *ZeroheavyDecoder) readOne() (byte, error) {
	ret := byte(0)
	for i := 0; i < 8; i++ {
		b, err := br.nextBit()
		if err != nil {
			return 0, err
		}
		ret <<= 1
		if b {
			ret |= 1
		}
	}
	return ret, nil
}

func (br *ZeroheavyDecoder) Read(p []byte) (int, error) {
	for i := range p {
		b, err := br.readOne()
		if err != nil {
			return i, err
		}
		p[i] = b
	}
	return len(p), nil
}

func (br *ZeroheavyDecoder) pushBit(b bool) {
	if br.deferredZero {
		if b {
			br.numDeferredOnes++
		} else {
			br.buffer = append(br.buffer, false)
			for br.numDeferredOnes > 0 {
				br.buffer = append(br.buffer, true)
				br.numDeferredOnes--
			}
		}
	} else {
		if b {
			br.buffer = append(br.buffer, true)
		} else {
			br.deferredZero = true
		}
	}
}

func (br *ZeroheavyDecoder) push7Bits(b byte) {
	for i := 0; i < 7; i++ {
		br.pushBit(b&(1<<(6-i)) != 0)
	}
}

func (br *ZeroheavyDecoder) nextBit() (bool, error) {
	for len(br.buffer) == 0 {
		if br.eofAfterBuffer {
			return false, io.EOF
		}
		br.eofAfterBuffer = br.refill()
	}
	ret := br.buffer[0]
	br.buffer = br.buffer[1:]
	return ret, nil
}

func (br *ZeroheavyDecoder) refill() bool {
	var buf [1]byte
	_, err := io.ReadFull(br.source, buf[:])
	if err != nil {
		return true
	}
	b := buf[0]
	if b == 0 {
		br.pushBit(false)
		br.pushBit(false)
	} else if b == 1 {
		br.pushBit(false)
		br.pushBit(true)
		for i := 0; i < 6; i++ {
			br.pushBit(false)
		}
	} else if b < 0x80 {
		br.pushBit(false)
		br.pushBit(true)
		br.push7Bits(b)
	} else {
		br.pushBit(true)
		br.push7Bits(b & 0x7f)
	}
	return false
}
