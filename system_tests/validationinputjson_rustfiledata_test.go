package arbtest

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/validator"
	"github.com/offchainlabs/nitro/validator/server_api"
	"github.com/offchainlabs/nitro/validator/server_arb"
)

func getWriteDataFromRustSide(t *testing.T, inputJSON *server_api.InputJSON) []byte {
	t.Helper()
	dir := t.TempDir()

	readData, err := inputJSON.Marshal()
	Require(t, err)
	readPath := filepath.Join(dir, fmt.Sprintf("block_inputs_%d_read.json", inputJSON.Id))
	Require(t, os.WriteFile(readPath, readData, 0600))

	writePath := filepath.Join(dir, fmt.Sprintf("block_inputs_%d_write.json", inputJSON.Id))
	Require(t, server_arb.DeserializeAndSerializeFileData(readPath, writePath))
	writeData, err := os.ReadFile(writePath)
	Require(t, err)

	return writeData
}

func TestGoInputJSONRustFileDataRoundtripWithoutUserWasms(t *testing.T) {
	preimages := make(map[arbutil.PreimageType]map[common.Hash][]byte)
	preimages[arbutil.Keccak256PreimageType] = make(map[common.Hash][]byte)
	preimages[arbutil.Keccak256PreimageType][common.MaxHash] = []byte{1}

	// Don't include DebugChain as it isn't used on rust side
	sampleValidationInput := &validator.ValidationInput{
		Id:            1,
		HasDelayedMsg: true,
		DelayedMsgNr:  2,
		Preimages:     preimages,
		BatchInfo:     []validator.BatchInfo{{Number: 3}},
		DelayedMsg:    []byte{4},
		StartState: validator.GoGlobalState{
			BlockHash:  common.MaxHash,
			SendRoot:   common.MaxHash,
			Batch:      5,
			PosInBatch: 6,
		},
	}
	sampleValidationInputJSON := server_api.ValidationInputToJson(sampleValidationInput)
	writeData := getWriteDataFromRustSide(t, sampleValidationInputJSON)

	var resWithoutUserWasms server_api.InputJSON
	Require(t, json.Unmarshal(writeData, &resWithoutUserWasms))
	if !reflect.DeepEqual(*sampleValidationInputJSON, resWithoutUserWasms) {
		t.Fatal("ValidationInputJSON without UserWasms, mismatch on rust and go side")
	}

}

type inputJSONWithUserWasmsOnly struct {
	UserWasms map[ethdb.WasmTarget]map[common.Hash][]byte
}

// UnmarshalJSON is a custom function defined to encapsulate how UserWasms are handled on the rust side.
// When ValidationInputToJson is called on ValidationInput, it compresses the wasm data byte array and
// then encodes this to a base64 string, this when deserialized on the rust side through FileData- the
// compressed data is first uncompressed and also the module hash (Bytes32) is read without the 0x prefix,
// so we define a custom UnmarshalJSON to extract UserWasms map from the data written by rust side.
func (u *inputJSONWithUserWasmsOnly) UnmarshalJSON(data []byte) error {
	type rawUserWasms struct {
		UserWasms map[ethdb.WasmTarget]map[string]string
	}
	var rawUWs rawUserWasms
	if err := json.Unmarshal(data, &rawUWs); err != nil {
		return err
	}
	tmp := make(map[ethdb.WasmTarget]map[common.Hash][]byte)
	for wasmTarget, innerMap := range rawUWs.UserWasms {
		tmp[wasmTarget] = make(map[common.Hash][]byte)
		for hashKey, value := range innerMap {
			valBytes, err := base64.StdEncoding.DecodeString(value)
			if err != nil {
				return err
			}
			tmp[wasmTarget][common.HexToHash("0x"+hashKey)] = valBytes
		}
	}
	u.UserWasms = tmp
	return nil
}

func TestGoInputJSONRustFileDataRoundtripWithUserWasms(t *testing.T) {
	userWasms := make(map[ethdb.WasmTarget]map[common.Hash][]byte)
	userWasms["arch1"] = make(map[common.Hash][]byte)
	userWasms["arch1"][common.MaxHash] = []byte{2}

	// Don't include DebugChain as it isn't used on rust side
	sampleValidationInput := &validator.ValidationInput{
		Id:        1,
		UserWasms: userWasms,
		BatchInfo: []validator.BatchInfo{{Number: 1}}, // This needs to be set for FileData to successfully deserialize, else it errors for invalid type null
	}
	sampleValidationInputJSON := server_api.ValidationInputToJson(sampleValidationInput)
	writeData := getWriteDataFromRustSide(t, sampleValidationInputJSON)

	var resUserWasmsOnly inputJSONWithUserWasmsOnly
	Require(t, json.Unmarshal(writeData, &resUserWasmsOnly))
	if !reflect.DeepEqual(userWasms, resUserWasmsOnly.UserWasms) {
		t.Fatal("ValidationInputJSON with UserWasms only, mismatch on rust and go side")
	}
}
