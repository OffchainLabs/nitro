// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE
package challengecache

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	l2stateprovider "github.com/OffchainLabs/bold/layer2-state-provider"
	state_hashes "github.com/OffchainLabs/bold/state-commitments/state-hashes"
	"github.com/ethereum/go-ethereum/common"
)

var _ HistoryCommitmentCacher = (*Cache)(nil)

func TestCache(t *testing.T) {
	basePath := t.TempDir()
	if err := os.MkdirAll(basePath, os.ModePerm); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(basePath); err != nil {
			t.Fatal(err)
		}
	})
	cache := New(basePath)
	key := &Key{
		WavmModuleRoot: common.BytesToHash([]byte("foo")),
		MessageHeight:  0,
		StepHeights:    []l2stateprovider.Height{l2stateprovider.Height(0)},
	}
	t.Run("Not found", func(t *testing.T) {
		_, err := cache.Get(key, 0)
		if !errors.Is(err, ErrNotFoundInCache) {
			t.Fatal(err)
		}
	})
	t.Run("Putting empty root fails", func(t *testing.T) {
		if err := cache.Put(key, state_hashes.NewStateHashes([]common.Hash{}, 0)); !errors.Is(err, ErrNoStateRoots) {
			t.Fatalf("Unexpected error: %v", err)
		}
	})
	wantHashes := []common.Hash{
		common.BytesToHash([]byte("foo")),
		common.BytesToHash([]byte("bar")),
		common.BytesToHash([]byte("baz")),
	}
	want := state_hashes.NewStateHashes(wantHashes, uint64(len(wantHashes)))
	err := cache.Put(key, want)
	if err != nil {
		t.Fatal(err)
	}
	got, err := cache.Get(key, 3)
	if err != nil {
		t.Fatal(err)
	}
	if got.Length() != want.Length() {
		t.Fatalf("Wrong number of roots. Expected %d, got %d", want.Length(), got.Length())
	}
	for i := uint64(0); i < got.Length(); i++ {
		if got.At(i) != want.At(i) {
			t.Fatalf("Wrong root. Expected %#x, got %#x", want.At(i), got.At(i))
		}
	}
}

func TestReadWriteStateRoots(t *testing.T) {
	t.Run("read up to, but had empty reader", func(t *testing.T) {
		b := bytes.NewBuffer([]byte{})
		_, err := readStateRoots(b, 100)
		if err == nil {
			t.Fatal("Wanted error")
		}
		if !strings.Contains(err.Error(), "only read 0 state roots") {
			t.Fatal("Unexpected error")
		}
	})
	t.Run("read single root", func(t *testing.T) {
		b := bytes.NewBuffer([]byte{})
		want := common.BytesToHash([]byte("foo"))
		b.Write(want.Bytes())
		roots, err := readStateRoots(b, 1)
		if err != nil {
			t.Fatal(err)
		}
		if roots.Length() == 0 {
			t.Fatal("Got no roots")
		}
		if roots.At(0) != want {
			t.Fatalf("Wrong root. Expected %#x, got %#x", want, roots.At(0))
		}
	})
	t.Run("Three roots exist, want to read only two", func(t *testing.T) {
		b := bytes.NewBuffer([]byte{})
		foo := common.BytesToHash([]byte("foo"))
		bar := common.BytesToHash([]byte("bar"))
		baz := common.BytesToHash([]byte("baz"))
		b.Write(foo.Bytes())
		b.Write(bar.Bytes())
		b.Write(baz.Bytes())
		roots, err := readStateRoots(b, 2)
		if err != nil {
			t.Fatal(err)
		}
		if roots.Length() != 2 {
			t.Fatalf("Expected two roots, got %d", roots.Length())
		}
		if roots.At(0) != foo {
			t.Fatalf("Wrong root. Expected %#x, got %#x", foo, roots.At(0))
		}
		if roots.At(1) != bar {
			t.Fatalf("Wrong root. Expected %#x, got %#x", bar, roots.At(1))
		}
	})
	t.Run("Fails to write enough data to writer", func(t *testing.T) {
		m := &mockWriter{wantErr: true}
		err := writeStateRoots(m, state_hashes.NewStateHashes([]common.Hash{common.BytesToHash([]byte("foo"))}, 1))
		if err == nil {
			t.Fatal("Wanted error")
		}
		m = &mockWriter{wantErr: false, numWritten: 16}
		err = writeStateRoots(m, state_hashes.NewStateHashes([]common.Hash{common.BytesToHash([]byte("foo"))}, 1))
		if err == nil {
			t.Fatal("Wanted error")
		}
		if !strings.Contains(err.Error(), "expected to write 32 bytes") {
			t.Fatalf("Got wrong error kind: %v", err)
		}
	})
}

type mockWriter struct {
	wantErr    bool
	numWritten int
}

func (m *mockWriter) Write(_ []byte) (n int, err error) {
	if m.wantErr {
		return 0, errors.New("something went wrong")
	}
	return m.numWritten, nil
}

type mockReader struct {
	wantErr   bool
	err       error
	roots     []common.Hash
	readIdx   int
	bytesRead int
}

func (m *mockReader) Read(out []byte) (n int, err error) {
	if m.wantErr {
		return 0, m.err
	}
	if m.readIdx == len(m.roots) {
		return 0, io.EOF
	}
	copy(out, m.roots[m.readIdx].Bytes())
	m.readIdx++
	return m.bytesRead, nil
}

func Test_readStateRoots(t *testing.T) {
	t.Run("Unexpected error", func(t *testing.T) {
		want := []common.Hash{
			common.BytesToHash([]byte("foo")),
			common.BytesToHash([]byte("bar")),
			common.BytesToHash([]byte("baz")),
		}
		m := &mockReader{wantErr: true, roots: want, err: errors.New("foo")}
		_, err := readStateRoots(m, 1)
		if err == nil {
			t.Fatal(err)
		}
		if !strings.Contains(err.Error(), "foo") {
			t.Fatalf("Unexpected error: %v", err)
		}
	})
	t.Run("EOF, but did not read as much as was expected", func(t *testing.T) {
		want := []common.Hash{
			common.BytesToHash([]byte("foo")),
			common.BytesToHash([]byte("bar")),
			common.BytesToHash([]byte("baz")),
		}
		m := &mockReader{wantErr: true, roots: want, err: io.EOF}
		_, err := readStateRoots(m, 100)
		if err == nil {
			t.Fatal(err)
		}
		if !strings.Contains(err.Error(), "wanted to read 100") {
			t.Fatalf("Unexpected error: %v", err)
		}
	})
	t.Run("Reads wrong number of bytes", func(t *testing.T) {
		want := []common.Hash{
			common.BytesToHash([]byte("foo")),
			common.BytesToHash([]byte("bar")),
			common.BytesToHash([]byte("baz")),
		}
		m := &mockReader{wantErr: false, roots: want, bytesRead: 16}
		_, err := readStateRoots(m, 2)
		if err == nil {
			t.Fatal(err)
		}
		if !strings.Contains(err.Error(), "expected to read 32 bytes, got 16") {
			t.Fatalf("Unexpected error: %v", err)
		}
	})
	t.Run("Reads all until EOF", func(t *testing.T) {
		want := []common.Hash{
			common.BytesToHash([]byte("foo")),
			common.BytesToHash([]byte("bar")),
			common.BytesToHash([]byte("baz")),
		}
		m := &mockReader{wantErr: false, roots: want, bytesRead: 32}
		got, err := readStateRoots(m, 3)
		if err != nil {
			t.Fatal(err)
		}
		if uint64(len(want)) != got.Length() {
			t.Fatal("Wrong number of roots")
		}
		for i := uint64(0); i < got.Length(); i++ {
			if got.At(i) != want[i] {
				t.Fatal("Wrong root")
			}
		}
	})
}

func Test_determineFilePath(t *testing.T) {
	type args struct {
		baseDir string
		key     *Key
	}
	tests := []struct {
		name        string
		args        args
		want        string
		wantErr     bool
		errContains string
	}{
		{
			name: "OK",
			args: args{
				baseDir: "",
				key: &Key{
					MessageHeight: 100,
					StepHeights:   []l2stateprovider.Height{l2stateprovider.Height(50)},
				},
			},
			want:    "wavm-module-root-0x0000000000000000000000000000000000000000000000000000000000000000/message-num-100/subchallenge-level-1-big-step-50/state-roots",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := determineFilePath(tt.args.baseDir, tt.args.key)
			if (err != nil) != tt.wantErr {
				t.Logf("got: %v, and key %+v, got %s", err, tt.args.key, got)
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Fatalf("Expected %s, got %s", tt.errContains, err.Error())
				}
				t.Errorf("determineFilePath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf(
					"determineFilePath() = %v, want %v",
					got,
					tt.want,
				)
			}
		})
	}
}

func BenchmarkCache_Read_32Mb(b *testing.B) {
	b.StopTimer()
	basePath := os.TempDir()
	if err := os.MkdirAll(basePath, os.ModePerm); err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() {
		if err := os.RemoveAll(basePath); err != nil {
			b.Fatal(err)
		}
	})
	cache := New(basePath)
	key := &Key{
		WavmModuleRoot: common.BytesToHash([]byte("foo")),
		MessageHeight:  0,
		StepHeights:    []l2stateprovider.Height{l2stateprovider.Height(0)},
	}
	numRoots := 1 << 20
	rootsHashes := make([]common.Hash, numRoots)
	for i := range rootsHashes {
		rootsHashes[i] = common.BytesToHash([]byte(fmt.Sprintf("%d", i)))
	}
	roots := state_hashes.NewStateHashes(rootsHashes, uint64(len(rootsHashes)))
	if err := cache.Put(key, roots); err != nil {
		b.Fatal(err)
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		readUpTo := uint64(1 << 20)
		roots, err := cache.Get(key, readUpTo)
		if err != nil {
			b.Fatal(err)
		}
		if roots.Length() != uint64(numRoots) {
			b.Fatalf("Wrong number of roots. Expected %d, got %d", numRoots, roots.Length())
		}
	}
}
