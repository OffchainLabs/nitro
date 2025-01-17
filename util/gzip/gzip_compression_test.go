package gzip

import (
	"bytes"
	"testing"
)

func TestCompressDecompress(t *testing.T) {
	sampleData := []byte{1, 2, 3, 4}
	compressedData, err := CompressGzip(sampleData)
	if err != nil {
		t.Fatalf("got error gzip-compressing data: %v", err)
	}
	gotData, err := DecompressGzip(compressedData)
	if err != nil {
		t.Fatalf("got error gzip-decompressing data: %v", err)
	}
	if !bytes.Equal(sampleData, gotData) {
		t.Fatal("original data and decompression of its compression don't match")
	}
}
