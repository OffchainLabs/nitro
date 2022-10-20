package arbnode

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

func EncodeBlobs(data []byte) types.Blobs {
	blobs := []types.Blob{{}}
	blobIndex := 0
	fieldIndex := -1
	for i := 0; i < len(data); i += 31 {
		fieldIndex++
		if fieldIndex == params.FieldElementsPerBlob {
			blobs = append(blobs, types.Blob{})
			blobIndex++
			fieldIndex = 0
		}
		max := i + 31
		if max > len(data) {
			max = len(data)
		}
		copy(blobs[blobIndex][fieldIndex][:], data[i:max])
	}
	return blobs
}

func DecodeBlob(blob [][]byte) []byte {
	var data []byte
	for _, b := range blob {
		data = append(data, b[0:31]...)
	}
	// XXX: the following removes trailing 0s, which could be unexpected for certain blobs
	i := len(data) - 1
	for ; i >= 0; i-- {
		if data[i] != 0x00 {
			break
		}
	}
	data = data[:i+1]
	return data
}

func DecodeUint256String(hexOrDecimal string) (*uint256.Int, error) {
	var base = 10
	if strings.HasPrefix(hexOrDecimal, "0x") {
		base = 16
	}
	b, ok := new(big.Int).SetString(hexOrDecimal, base)
	if !ok {
		return nil, fmt.Errorf("invalid value")
	}
	val256, nok := uint256.FromBig(b)
	if nok {
		return nil, fmt.Errorf("value is too big")
	}
	return val256, nil
}
