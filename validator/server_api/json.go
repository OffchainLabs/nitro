package server_api

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/validator"
)

type PreimagesMapJson struct {
	Map map[common.Hash][]byte
}

func (m *PreimagesMapJson) MarshalJSON() ([]byte, error) {
	size := 2 // {}
	encoding := base64.StdEncoding
	for key, value := range m.Map {
		size += 5 // "":""
		size += encoding.EncodedLen(len(key))
		size += encoding.EncodedLen(len(value))
	}
	if len(m.Map) > 0 {
		// commas
		size += len(m.Map) - 1
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

type BatchInfoJson struct {
	Number  uint64
	DataB64 string
}

type ValidationInputJson struct {
	Id            uint64
	HasDelayedMsg bool
	DelayedMsgNr  uint64
	PreimagesB64  PreimagesMapJson
	BatchInfo     []BatchInfoJson
	DelayedMsgB64 string
	StartState    validator.GoGlobalState
}

func ValidationInputToJson(entry *validator.ValidationInput) *ValidationInputJson {
	res := &ValidationInputJson{
		Id:            entry.Id,
		HasDelayedMsg: entry.HasDelayedMsg,
		DelayedMsgNr:  entry.DelayedMsgNr,
		DelayedMsgB64: base64.StdEncoding.EncodeToString(entry.DelayedMsg),
		StartState:    entry.StartState,
		PreimagesB64:  PreimagesMapJson{entry.Preimages},
	}
	for _, binfo := range entry.BatchInfo {
		encData := base64.StdEncoding.EncodeToString(binfo.Data)
		res.BatchInfo = append(res.BatchInfo, BatchInfoJson{binfo.Number, encData})
	}
	return res
}

func ValidationInputFromJson(entry *ValidationInputJson) (*validator.ValidationInput, error) {
	valInput := &validator.ValidationInput{
		Id:            entry.Id,
		HasDelayedMsg: entry.HasDelayedMsg,
		DelayedMsgNr:  entry.DelayedMsgNr,
		StartState:    entry.StartState,
		Preimages:     entry.PreimagesB64.Map,
	}
	delayed, err := base64.StdEncoding.DecodeString(entry.DelayedMsgB64)
	if err != nil {
		return nil, err
	}
	valInput.DelayedMsg = delayed
	for _, binfo := range entry.BatchInfo {
		data, err := base64.StdEncoding.DecodeString(binfo.DataB64)
		if err != nil {
			return nil, err
		}
		decInfo := validator.BatchInfo{
			Number: binfo.Number,
			Data:   data,
		}
		valInput.BatchInfo = append(valInput.BatchInfo, decInfo)
	}
	return valInput, nil
}

type MachineStepResultJson struct {
	Hash        common.Hash
	Position    uint64
	Status      uint8
	GlobalState validator.GoGlobalState
}

func MachineStepResultToJson(result *validator.MachineStepResult) *MachineStepResultJson {
	return &MachineStepResultJson{
		Hash:        result.Hash,
		Position:    result.Position,
		Status:      uint8(result.Status),
		GlobalState: result.GlobalState,
	}
}

func MachineStepResultFromJson(resultJson *MachineStepResultJson) (*validator.MachineStepResult, error) {

	return &validator.MachineStepResult{
		Hash:        resultJson.Hash,
		Position:    resultJson.Position,
		Status:      validator.MachineStatus(resultJson.Status),
		GlobalState: resultJson.GlobalState,
	}, nil
}
