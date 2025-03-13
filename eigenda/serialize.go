package eigenda

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"

	"github.com/Layr-Labs/eigenda/encoding"
	"github.com/Layr-Labs/eigenda/encoding/rs"
	"github.com/Layr-Labs/eigenda/encoding/utils/codec"
	"github.com/consensys/gnark-crypto/ecc/bn254/fr"
)

/*
	These decodings are translated directly from core EigenDA default client codec:
	- https://github.com/Layr-Labs/eigenda/blob/44569ec461c9a1dd1191e7999a72e63bd1e7aba9/api/clients/codecs/ifft_codec.go#L27-L38
*/

func GenericDecodeBlob(data []byte) ([]byte, error) {
	if len(data) <= 32 {
		return nil, fmt.Errorf("data is not of length greater than 32 bytes: %d", len(data))
	}

	data, err := decodeBlob(data)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func decodeBlob(data []byte) ([]byte, error) {
	length := binary.BigEndian.Uint32(data[2:6])

	// decode raw data modulo bn254
	decodedData := codec.RemoveEmptyByteFromPaddedBytes(data[32:])

	// get non blob header data
	reader := bytes.NewReader(decodedData)
	rawData := make([]byte, length)
	n, err := reader.Read(rawData)
	if err != nil {
		return nil, fmt.Errorf("failed to copy unpadded data into final buffer, length: %d, bytes read: %d", length, n)
	}
	if uint32(n) != length {
		return nil, fmt.Errorf("data length does not match length prefix")
	}

	return rawData, nil

}

func GenericEncodeBlob(data []byte) ([]byte, error) {
	var err error
	data, err = encodeBlob(data)
	if err != nil {
		return nil, fmt.Errorf("error encoding data: %w", err)
	}

	return padPow2(data)
}

func encodeBlob(rawData []byte) ([]byte, error) {
	if len(rawData) > math.MaxUint32 {
		return nil, fmt.Errorf("data length exceeds 2^32 bytes: %d", len(rawData))
	}

	codecBlobHeader := make([]byte, 32)
	// first byte is always 0 to ensure the codecBlobHeader is a valid bn254 element
	// encode version byte
	codecBlobHeader[1] = byte(0x0)

	// encode length as uint32
	binary.BigEndian.PutUint32(codecBlobHeader[2:6], uint32(len(rawData))) // uint32 should be more than enough to store the length (approx 4gb)

	// encode raw data modulo bn254
	rawDataPadded := codec.ConvertByPaddingEmptyByte(rawData)

	// append raw data; reassign avoids copying
	encodedData := codecBlobHeader
	encodedData = append(encodedData, rawDataPadded...)

	return encodedData, nil
}

// pad data to the next power of 2
func padPow2(data []byte) ([]byte, error) {
	dataFr, err := rs.ToFrArray(data)
	if err != nil {
		return nil, fmt.Errorf("error converting data to fr.Element: %w", err)
	}

	dataFrLen := len(dataFr)
	dataFrLenPow2 := encoding.NextPowerOf2(uint64(dataFrLen))

	// expand data to the next power of 2
	paddedDataFr := make([]fr.Element, dataFrLenPow2)
	for i := 0; i < len(paddedDataFr); i++ {
		if i < len(dataFr) {
			paddedDataFr[i].Set(&dataFr[i])
		} else {
			paddedDataFr[i].SetZero()
		}
	}

	return rs.ToByteArray(paddedDataFr, dataFrLenPow2*encoding.BYTES_PER_SYMBOL), nil
}

// removeZeroPadding32Bytes removes any prefix padded zero bytes from an assumed
// 32 byte value
func removeZeroPadding32Bytes(arr []byte) ([]byte, error) {
	if len(arr) < 32 {
		return nil, fmt.Errorf("expected value >= 32 bytes; got %d", len(arr))
	}

	// iterate over prefix bytes and verify only zero's are included
	start := 0
	for start < len(arr)-32 {
		if arr[start] != 0x0 {
			return nil, fmt.Errorf("expecting only 0x0 prefixes, got %d at index %d", byte(arr[start]), start)
		}

		start++
	}

	// Ensure we return exactly 32 bytes
	end := start + 32
	if end > len(arr) {
		return nil, fmt.Errorf("unexpected error, computed range out of bounds")
	}

	return arr[start:end], nil
}
