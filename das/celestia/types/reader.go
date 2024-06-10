package types

import (
	"bytes"
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	daprovider "github.com/offchainlabs/nitro/arbstate/daprovider"
	"github.com/offchainlabs/nitro/das/celestia/tree"
)

func NewReaderForCelestia(celestiaReader CelestiaReader) *readerForCelestia {
	return &readerForCelestia{celestiaReader: celestiaReader}
}

type readerForCelestia struct {
	celestiaReader CelestiaReader
}

func (c *readerForCelestia) IsValidHeaderByte(headerByte byte) bool {
	return IsCelestiaMessageHeaderByte(headerByte)
}

// CelestiaMessageHeaderFlag indicates that this data is a Blob Pointer
// which will be used to retrieve data from Celestia
const CelestiaMessageHeaderFlag byte = 0x63

func hasBits(checking byte, bits byte) bool {
	return (checking & bits) == bits
}

func IsCelestiaMessageHeaderByte(header byte) bool {
	return hasBits(header, CelestiaMessageHeaderFlag)
}

func (c *readerForCelestia) RecoverPayloadFromBatch(
	ctx context.Context,
	batchNum uint64,
	batchBlockHash common.Hash,
	sequencerMsg []byte,
	preimageRecorder daprovider.PreimageRecorder,
	validateSeqMsg bool,
) ([]byte, error) {
	return RecoverPayloadFromCelestiaBatch(ctx, batchNum, sequencerMsg, c.celestiaReader, preimageRecorder, validateSeqMsg)
}

func RecoverPayloadFromCelestiaBatch(
	ctx context.Context,
	batchNum uint64,
	sequencerMsg []byte,
	celestiaReader CelestiaReader,
	preimageRecorder daprovider.PreimageRecorder,
	validateSeqMsg bool,
) ([]byte, error) {
	buf := bytes.NewBuffer(sequencerMsg[40:])

	header, err := buf.ReadByte()
	if err != nil {
		log.Error("Couldn't deserialize Celestia header byte", "err", err)
		return nil, nil
	}
	if !IsCelestiaMessageHeaderByte(header) {
		log.Error("Couldn't deserialize Celestia header byte", "err", errors.New("tried to deserialize a message that doesn't have the Celestia header"))
		return nil, nil
	}

	blobPointer := BlobPointer{}
	blobBytes := buf.Bytes()
	err = blobPointer.UnmarshalBinary(blobBytes)
	if err != nil {
		log.Error("Couldn't unmarshal Celestia blob pointer", "err", err)
		return nil, nil
	}

	payload, squareData, err := celestiaReader.Read(ctx, &blobPointer)
	if err != nil {
		log.Error("Failed to resolve blob pointer from celestia", "err", err)
		return nil, err
	}

	// we read a batch that is to be discarded, so we return the empty batch
	if len(payload) == 0 {
		return payload, nil
	}

	if preimageRecorder != nil {
		if squareData == nil {
			log.Error("squareData is nil, read from replay binary, but preimages are empty")
			return nil, err
		}

		odsSize := squareData.SquareSize / 2
		rowIndex := squareData.StartRow
		for _, row := range squareData.Rows {
			treeConstructor := tree.NewConstructor(preimageRecorder, odsSize)
			root, err := tree.ComputeNmtRoot(treeConstructor, uint(rowIndex), row)
			if err != nil {
				log.Error("Failed to compute row root", "err", err)
				return nil, err
			}

			rowRootMatches := bytes.Equal(squareData.RowRoots[rowIndex], root)
			if !rowRootMatches {
				log.Error("Row roots do not match", "eds row root", squareData.RowRoots[rowIndex], "calculated", root)
				log.Error("Row roots", "row_roots", squareData.RowRoots)
				return nil, err
			}
			rowIndex += 1
		}

		rowsCount := len(squareData.RowRoots)
		slices := make([][]byte, rowsCount+rowsCount)
		copy(slices[0:rowsCount], squareData.RowRoots)
		copy(slices[rowsCount:], squareData.ColumnRoots)

		dataRoot := tree.HashFromByteSlices(preimageRecorder, slices)

		dataRootMatches := bytes.Equal(dataRoot, blobPointer.DataRoot[:])
		if !dataRootMatches {
			log.Error("Data Root do not match", "blobPointer data root", blobPointer.DataRoot, "calculated", dataRoot)
			return nil, nil
		}
	}

	return payload, nil
}
