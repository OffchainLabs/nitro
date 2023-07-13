package execution

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util/jsonapi"
)

type JsonRecordResult struct {
	Pos       arbutil.MessageIndex
	BlockHash common.Hash
	Preimages jsonapi.PreimagesMapJson
}

func NewJsonRecordResult(result *RecordResult) *JsonRecordResult {
	return &JsonRecordResult{
		Pos:       result.Pos,
		BlockHash: result.BlockHash,
		Preimages: jsonapi.NewPreimagesMapJson(result.Preimages),
	}
}

func (j *JsonRecordResult) ToResult() *RecordResult {
	if j == nil {
		return nil
	}
	return &RecordResult{
		Pos:       j.Pos,
		BlockHash: j.BlockHash,
		Preimages: j.Preimages.Map,
	}
}
