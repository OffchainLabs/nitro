package gzip

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
)

func CompressGzip(data []byte) ([]byte, error) {
	var buffer bytes.Buffer
	gzipWriter := gzip.NewWriter(&buffer)
	if _, err := gzipWriter.Write(data); err != nil {
		return nil, fmt.Errorf("failed to write to gzip writer: %w", err)
	}
	if err := gzipWriter.Close(); err != nil {
		return nil, fmt.Errorf("failed to close gzip writer: %w", err)
	}
	return buffer.Bytes(), nil
}

func DecompressGzip(data []byte) ([]byte, error) {
	buffer := bytes.NewReader(data)
	gzipReader, err := gzip.NewReader(buffer)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzipReader.Close()
	decompressData, err := io.ReadAll(gzipReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read decompressed data: %w", err)
	}
	return decompressData, nil
}
