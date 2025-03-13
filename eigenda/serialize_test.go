package eigenda

import (
	"bytes"
	"testing"
)

func Test_EncodeDecodeBlob(t *testing.T) {
	rawBlob := []byte("optimistic nihilism")

	encodedBlob, err := GenericEncodeBlob(rawBlob)
	if err != nil {
		t.Fatalf("failed to encode blob: %v", err)
	}

	decodedBlob, err := GenericDecodeBlob(encodedBlob)
	if err != nil {
		t.Fatalf("failed to decode blob: %v", err)
	}

	if string(decodedBlob) != string(rawBlob) {
		t.Fatalf("decoded blob does not match raw blob")
	}
}

func Test_RemoveZeroPadding32Bytes(t *testing.T) {
	testArr := make([]byte, 32)
	for i := range 32 {
		testArr[i] = byte(i)
	}

	// 1 - do nothing
	out1, err := removeZeroPadding32Bytes(testArr)
	if err != nil {
		t.Fatalf("failed to sanitize bytes to field element: %v", testArr)
	}

	if !bytes.Equal(testArr, out1) {
		t.Fatalf("not equal; in %v, out %v", testArr, out1)
	}

	// 2 - add padding and ensure its been removed
	testArr = append([]byte{0x0, 0x0, 0x0}, testArr...)

	out2, err := removeZeroPadding32Bytes(testArr)

	if !bytes.Equal(out1, out2) {
		t.Fatalf("not equal; in %v, out %v", out1, out2)
	}

	// 3 - pad nonzero and ensure error

	testArr = append([]byte{0x69}, testArr...)

	_, err = removeZeroPadding32Bytes(testArr)
	if err == nil {
		t.Fatalf("expected error: %v", err)
	}

	// 4 - ensure error when input too small

	_, err = removeZeroPadding32Bytes([]byte{0x42})
	if err == nil {
		t.Fatalf("expected error: %v", err)
	}
}
