// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package jsonapi

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/common"
)

type PreimagesMapJson struct {
	Map map[common.Hash][]byte
}

func NewPreimagesMapJson(inner map[common.Hash][]byte) *PreimagesMapJson {
	return &PreimagesMapJson{inner}
}

func (m *PreimagesMapJson) MarshalJSON() ([]byte, error) {
	encoding := base64.StdEncoding
	size := 2                                          // {}
	size += (5 + encoding.EncodedLen(32)) * len(m.Map) // "000..000":""
	if len(m.Map) > 0 {
		size += len(m.Map) - 1 // commas
	}
	for _, value := range m.Map {
		size += encoding.EncodedLen(len(value))
	}
	out := make([]byte, size)
	i := 0
	out[i] = '{'
	i++
	for key, value := range m.Map {
		if i > 1 {
			out[i] = ','
			i++
		}
		out[i] = '"'
		i++
		encoding.Encode(out[i:], key[:])
		i += encoding.EncodedLen(len(key))
		out[i] = '"'
		i++
		out[i] = ':'
		i++
		out[i] = '"'
		i++
		encoding.Encode(out[i:], value)
		i += encoding.EncodedLen(len(value))
		out[i] = '"'
		i++
	}
	out[i] = '}'
	i++
	if i != len(out) {
		return nil, fmt.Errorf("preimage map wrote %v bytes but expected to write %v", i, len(out))
	}
	return out, nil
}

func readNonWhitespace(data *[]byte) (byte, error) {
	c := byte('\t')
	for c == '\t' || c == '\n' || c == '\v' || c == '\f' || c == '\r' || c == ' ' {
		if len(*data) == 0 {
			return 0, io.ErrUnexpectedEOF
		}
		c = (*data)[0]
		*data = (*data)[1:]
	}
	return c, nil
}

func expectCharacter(data *[]byte, expected rune) error {
	got, err := readNonWhitespace(data)
	if err != nil {
		return fmt.Errorf("while looking for '%v' got %w", expected, err)
	}
	if rune(got) != expected {
		return fmt.Errorf("while looking for '%v' got '%v'", expected, rune(got))
	}
	return nil
}

func getStrLen(data []byte) (int, error) {
	// We don't allow strings to contain an escape sequence.
	// Searching for a backslash here would be duplicated work.
	// If the returned string length includes a backslash, base64 decoding will fail and error there.
	strLen := bytes.IndexByte(data, '"')
	if strLen == -1 {
		return 0, fmt.Errorf("%w: hit end of preimages map looking for end quote", io.ErrUnexpectedEOF)
	}
	return strLen, nil
}

func (m *PreimagesMapJson) UnmarshalJSON(data []byte) error {
	err := expectCharacter(&data, '{')
	if err != nil {
		return err
	}
	m.Map = make(map[common.Hash][]byte)
	encoding := base64.StdEncoding
	// Used to store base64 decoded data
	// Returned unmarshalled preimage slices will just be parts of this one
	buf := make([]byte, encoding.DecodedLen(len(data)))
	for {
		c, err := readNonWhitespace(&data)
		if err != nil {
			return fmt.Errorf("while looking for key in preimages map got %w", err)
		}
		if len(m.Map) == 0 && c == '}' {
			break
		} else if c != '"' {
			return fmt.Errorf("expected '\"' to begin key in preimages map but got '%v'", c)
		}
		strLen, err := getStrLen(data)
		if err != nil {
			return err
		}
		maxKeyLen := encoding.DecodedLen(strLen)
		if maxKeyLen > len(buf) {
			return fmt.Errorf("preimage key base64 possible length %v is greater than buffer size of %v", maxKeyLen, len(buf))
		}
		keyLen, err := encoding.Decode(buf, data[:strLen])
		if err != nil {
			return fmt.Errorf("error base64 decoding preimage key: %w", err)
		}
		var key common.Hash
		if keyLen != len(key) {
			return fmt.Errorf("expected preimage to be %v bytes long, but got %v bytes", len(key), keyLen)
		}
		copy(key[:], buf[:len(key)])
		// We don't need to advance buf here because we already copied the data we needed out of it
		data = data[strLen+1:]
		err = expectCharacter(&data, ':')
		if err != nil {
			return err
		}
		err = expectCharacter(&data, '"')
		if err != nil {
			return err
		}
		strLen, err = getStrLen(data)
		if err != nil {
			return err
		}
		maxValueLen := encoding.DecodedLen(strLen)
		if maxValueLen > len(buf) {
			return fmt.Errorf("preimage value base64 possible length %v is greater than buffer size of %v", maxValueLen, len(buf))
		}
		valueLen, err := encoding.Decode(buf, data[:strLen])
		if err != nil {
			return fmt.Errorf("error base64 decoding preimage value: %w", err)
		}
		m.Map[key] = buf[:valueLen]
		buf = buf[valueLen:]
		data = data[strLen+1:]
		c, err = readNonWhitespace(&data)
		if err != nil {
			return fmt.Errorf("after value in preimages map got %w", err)
		}
		if c == '}' {
			break
		} else if c != ',' {
			return fmt.Errorf("expected ',' or '}' after value in preimages map but got '%v'", c)
		}
	}
	return nil
}
