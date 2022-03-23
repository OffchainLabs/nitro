package zeroheavy

import (
	"bytes"
	"errors"
	"io"
	"math/rand"
	"testing"
)

func TestZeroheavyNullInput(t *testing.T) {
	source := bytes.NewReader([]byte{})
	enc := NewZeroheavyEncoder(source)
	dec := NewZeroheavyDecoder(enc)

	var buf [256]byte
	n, err := dec.Read(buf[:])
	if !errors.Is(err, io.EOF) {
		t.Fatal()
	}
	if n != 0 {
		t.Fatal(n, buf[0])
	}
}

func TestZeroHeavyOneByte(t *testing.T) {
	for i := 0; i < 256; i++ {
		source := bytes.NewReader([]byte{byte(i)})
		enc := NewZeroheavyEncoder(source)
		dec := NewZeroheavyDecoder(enc)

		buf, err := io.ReadAll(dec)
		if err != nil {
			t.Error(err)
		}
		if len(buf) != 1 {
			t.Fatal(i, len(buf))
		}
		if buf[0] != byte(i) {
			t.Fatal(buf[0], i)
		}
	}
}

func TestZeroHeavyRandomData(t *testing.T) {
	rand.Seed(0)
	for i := 0; i < 1024; i++ {
		size := rand.Uint64() % 4096
		buf := make([]byte, size)
		_, _ = rand.Read(buf)
		dec := NewZeroheavyDecoder(NewZeroheavyEncoder(bytes.NewReader(buf)))
		res, err := io.ReadAll(dec)
		if err != nil {
			t.Error(err)
		}
		if !bytes.Equal(buf, res) {
			t.Fatal()
		}
	}
}
