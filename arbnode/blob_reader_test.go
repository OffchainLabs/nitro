package arbnode

import (
	"encoding/json"
	"io"
	"os"
	"path"
	"reflect"
	"testing"

	"github.com/r3labs/diff/v3"
)

func TestSaveBlobsToDisk(t *testing.T) {
	response := []blobResponseItem{{
		BlockRoot:       "a",
		Index:           0,
		Slot:            5,
		BlockParentRoot: "a0",
		ProposerIndex:   9,
		Blob:            []byte{1},
		KzgCommitment:   []byte{1},
		KzgProof:        []byte{1},
	}, {
		BlockRoot:       "a",
		Index:           1,
		Slot:            5,
		BlockParentRoot: "a0",
		ProposerIndex:   10,
		Blob:            []byte{2},
		KzgCommitment:   []byte{2},
		KzgProof:        []byte{2},
	}}
	testDir := t.TempDir()
	err := saveBlobDataToDisk(response, 5, testDir)
	Require(t, err)

	filePath := path.Join(testDir, "5")
	file, err := os.Open(filePath)
	Require(t, err)
	defer file.Close()

	data, err := io.ReadAll(file)
	Require(t, err)
	var full fullResult[[]blobResponseItem]
	err = json.Unmarshal(data, &full)
	Require(t, err)
	if !reflect.DeepEqual(full.Data, response) {
		changelog, err := diff.Diff(full.Data, response)
		Require(t, err)
		Fail(t, "blob data saved to disk does not match actual blob data", changelog)
	}
}
