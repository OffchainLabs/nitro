package espresso

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"unicode/utf8"

	"github.com/ethereum/go-ethereum/crypto"
)

type Commitment [32]byte

func CommitmentFromUint256(n *U256) (Commitment, error) {
	var bytes [32]byte

	bigEndian := n.Bytes()
	if len(bigEndian) > 32 {
		return Commitment{}, fmt.Errorf("integer out of range for U256 (%d)", n)
	}

	// `n` might have fewer than 32 bytes, if the commitment starts with one or more zeros. Pad out
	// to 32 bytes exactly, adding zeros at the beginning to be consistent with big-endian byte
	// order.
	if len(bigEndian) < 32 {
		zeros := make([]byte, 32-len(bigEndian))
		bigEndian = append(zeros, bigEndian...)
	}

	for i, b := range bigEndian {
		// Bytes() returns the bytes in big endian order, but HotShot encodes commitments as
		// U256 in little endian order, so we populate the bytes in reverse order.
		bytes[31-i] = b
	}
	return bytes, nil
}

func (c Commitment) Uint256() *U256 {
	var bigEndian [32]byte
	for i, b := range c {
		// HotShot interprets the commitment as a little-endian integer. `SetBytes` takes the bytes
		// in big-endian order, so we populate the bytes in reverse order.
		bigEndian[31-i] = b
	}
	return NewU256().SetBytes(bigEndian)
}

func (c Commitment) Equals(other Commitment) bool {
	return bytes.Equal(c[:], other[:])
}

type RawCommitmentBuilder struct {
	hasher crypto.KeccakState
}

func NewRawCommitmentBuilder(name string) *RawCommitmentBuilder {
	b := new(RawCommitmentBuilder)
	b.hasher = crypto.NewKeccakState()
	return b.ConstantString(name)
}

// Append a constant string to the running hash.
//
// WARNING: The string `s` must be a constant. This function does not encode the length of `s` in
// the hash, which can lead to domain collisions when different strings with different lengths are
// used depending on the input object.
func (b *RawCommitmentBuilder) ConstantString(s string) *RawCommitmentBuilder {
	// The commitment scheme is only designed to work with UTF-8 strings. In the reference
	// implementation, written in Rust, all strings are UTF-8, but in Go we have to check.
	if !utf8.Valid([]byte(s)) {
		panic(fmt.Sprintf("ConstantString must only be called with valid UTF-8 strings: %v", s))
	}

	if _, err := io.WriteString(b.hasher, s); err != nil {
		panic(fmt.Sprintf("KeccakState Writer is not supposed to fail, but it did: %v", err))
	}

	// To denote the end of the string and act as a domain separator, include a byte sequence which
	// can never appear in a valid UTF-8 string.
	invalidUtf8 := []byte{0xC0, 0x7F}
	return b.FixedSizeBytes(invalidUtf8)
}

// Include a named field of another committable type.
func (b *RawCommitmentBuilder) Field(f string, c Commitment) *RawCommitmentBuilder {
	return b.ConstantString(f).FixedSizeBytes(c[:])
}

func (b *RawCommitmentBuilder) OptionalField(f string, c *Commitment) *RawCommitmentBuilder {
	b.ConstantString(f)

	// Encode a 0 or 1 to separate the nil domain from the non-nil domain.
	if c == nil {
		b.Uint64(0)
	} else {
		b.Uint64(1)
		b.FixedSizeBytes((*c)[:])
	}

	return b
}

// Include a named field of type `uint256` in the hash.
func (b *RawCommitmentBuilder) Uint256Field(f string, n *U256) *RawCommitmentBuilder {
	return b.ConstantString(f).Uint256(n)
}

// Include a value of type `uint256` in the hash.
func (b *RawCommitmentBuilder) Uint256(n *U256) *RawCommitmentBuilder {
	bytes := make([]byte, 32)
	n.FillBytes(bytes)

	// `FillBytes` uses big endian byte ordering, but the Espresso commitment scheme uses little
	// endian, so we need to reverse the bytes.
	for i, j := 0, len(bytes)-1; i < j; i, j = i+1, j-1 {
		bytes[i], bytes[j] = bytes[j], bytes[i]
	}

	return b.FixedSizeBytes(bytes)
}

// Include a named field of type `uint64` in the hash.
func (b *RawCommitmentBuilder) Uint64Field(f string, n uint64) *RawCommitmentBuilder {
	return b.ConstantString(f).Uint64(n)
}

// Include a value of type `uint64` in the hash.
func (b *RawCommitmentBuilder) Uint64(n uint64) *RawCommitmentBuilder {
	bytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(bytes, n)
	return b.FixedSizeBytes(bytes)
}

// Include a named field of fixed length in the hash.
//
// WARNING: Go's type system cannot express the requirement that `bytes` is a fixed size array of
// any size. The best we can do is take a dynamically sized slice. However, this function uses a
// fixed-size encoding; namely, it does not encode the length of `bytes` in the hash, which can lead
// to domain collisions when this function is called with a slice which can have different lengths
// depending on the input object.
//
// The caller must ensure that this function is only used with slices whose length is statically
// determined by the type being committed to.
func (b *RawCommitmentBuilder) FixedSizeField(f string, bytes Bytes) *RawCommitmentBuilder {
	return b.ConstantString(f).FixedSizeBytes(bytes)
}

// Append a fixed size byte array to the running hash.
//
// WARNING: Go's type system cannot express the requirement that `bytes` is a fixed size array of
// any size. The best we can do is take a dynamically sized slice. However, this function uses a
// fixed-size encoding; namely, it does not encode the length of `bytes` in the hash, which can lead
// to domain collisions when this function is called with a slice which can have different lengths
// depending on the input object.
//
// The caller must ensure that this function is only used with slices whose length is statically
// determined by the type being committed to.
func (b *RawCommitmentBuilder) FixedSizeBytes(bytes Bytes) *RawCommitmentBuilder {
	b.hasher.Write(bytes)
	return b
}

// Include a named field of dynamic length in the hash.
func (b *RawCommitmentBuilder) VarSizeField(f string, bytes Bytes) *RawCommitmentBuilder {
	return b.ConstantString(f).VarSizeBytes(bytes)
}

// Include a byte array whose length can be dynamic to the running hash.
func (b *RawCommitmentBuilder) VarSizeBytes(bytes Bytes) *RawCommitmentBuilder {
	// First commit to the length, to prevent length extension and domain collision attacks.
	b.Uint64(uint64(len(bytes)))
	b.hasher.Write(bytes)
	return b
}

func (b *RawCommitmentBuilder) Finalize() Commitment {
	var comm Commitment
	bytes := b.hasher.Sum(nil)
	copy(comm[:], bytes)
	return comm
}
