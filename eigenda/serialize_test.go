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

func Test_StripZeroPrefixAndEnsure32Bytes(t *testing.T) {
	testArr := make([]byte, 32)
	for i := range 32 {
		testArr[i] = byte(i)
	}

	// 1 - do nothing
	out1, err := stripZeroPrefixAndEnsure32Bytes(testArr)
	if err != nil {
		t.Fatalf("failed to sanitize bytes to field element: %v", testArr)
	}

	if !bytes.Equal(testArr, out1) {
		t.Fatalf("not equal; in %v, out %v", testArr, out1)
	}

	// 2 - add padding and ensure its been removed
	testArr = append([]byte{0x0, 0x0, 0x0}, testArr...)

	out2, err := stripZeroPrefixAndEnsure32Bytes(testArr)

	if !bytes.Equal(out1, out2) {
		t.Fatalf("not equal; in %v, out %v", out1, out2)
	}

	// 3 - pad nonzero and ensure error

	testArr = append([]byte{0x69}, testArr...)

	_, err = stripZeroPrefixAndEnsure32Bytes(testArr)
	if err == nil {
		t.Fatalf("expected error: %v", err)
	}

	// 4 - ensure padding when input too small

	out3, err := stripZeroPrefixAndEnsure32Bytes([]byte{0x42})
	if err != nil {
		t.Fatalf("expected error: %v", err)
	}

	if out3[31] != 0x42 {
		t.Fatalf("expected 0x42 as last value in 32 byte arr")
	}
}
