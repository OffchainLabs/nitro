// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE
package challengecache

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

var _ HistoryCommitmentCacher = (*Cache)(nil)

func TestCache(t *testing.T) {
	ctx := context.Background()
	basePath := t.TempDir()
	if err := os.MkdirAll(basePath, os.ModePerm); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(basePath); err != nil {
			t.Fatal(err)
		}
	})
	cache, err := New(basePath)
	if err != nil {
		t.Fatal(err)
	}
	if err = cache.Init(ctx); err != nil {
		t.Fatal(err)
	}
	key := &Key{
		WavmModuleRoot: common.BytesToHash([]byte("foo")),
		MessageHeight:  0,
		StepHeights:    []uint64{0},
	}
	t.Run("Not found", func(t *testing.T) {
		_, err := cache.Get(key, 0)
		if !errors.Is(err, ErrNotFoundInCache) {
			t.Fatal(err)
		}
	})
	t.Run("Putting empty hash fails", func(t *testing.T) {
		if err := cache.Put(key, []common.Hash{}); !errors.Is(err, ErrNoHashes) {
			t.Fatalf("Unexpected error: %v", err)
		}
	})
	want := []common.Hash{
		common.BytesToHash([]byte("foo")),
		common.BytesToHash([]byte("bar")),
		common.BytesToHash([]byte("baz")),
	}
	err = cache.Put(key, want)
	if err != nil {
		t.Fatal(err)
	}
	got, err := cache.Get(key, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(want) {
		t.Fatalf("Wrong number of hashes. Expected %d, got %d", len(want), len(got))
	}
	for i, rt := range got {
		if rt != want[i] {
			t.Fatalf("Wrong root. Expected %#x, got %#x", want[i], rt)
		}
	}
}

func TestPrune(t *testing.T) {
	ctx := context.Background()
	basePath := t.TempDir()
	if err := os.MkdirAll(basePath, os.ModePerm); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(basePath); err != nil {
			t.Fatal(err)
		}
	})
	cache, err := New(basePath)
	if err != nil {
		t.Fatal(err)
	}
	if err = cache.Init(ctx); err != nil {
		t.Fatal(err)
	}
	key := &Key{
		WavmModuleRoot: common.BytesToHash([]byte("foo")),
		MessageHeight:  20,
		StepHeights:    []uint64{0},
	}
	if _, err = cache.Get(key, 0); !errors.Is(err, ErrNotFoundInCache) {
		t.Fatal(err)
	}
	t.Run("pruning non-existent dirs does nothing", func(t *testing.T) {
		if err = cache.Prune(ctx, key.MessageHeight); err != nil {
			t.Error(err)
		}
	})
	t.Run("pruning single item", func(t *testing.T) {
		want := []common.Hash{
			common.BytesToHash([]byte("foo")),
			common.BytesToHash([]byte("bar")),
			common.BytesToHash([]byte("baz")),
		}
		err = cache.Put(key, want)
		if err != nil {
			t.Fatal(err)
		}
		items, err := cache.Get(key, 3)
		if err != nil {
			t.Fatal(err)
		}
		if len(items) != len(want) {
			t.Fatalf("Wrong number of hashes. Expected %d, got %d", len(want), len(items))
		}
		if err = cache.Prune(ctx, key.MessageHeight); err != nil {
			t.Error(err)
		}
		if _, err = cache.Get(key, 3); !errors.Is(err, ErrNotFoundInCache) {
			t.Error(err)
		}
	})
	t.Run("does not prune items with message number > N", func(t *testing.T) {
		want := []common.Hash{
			common.BytesToHash([]byte("foo")),
			common.BytesToHash([]byte("bar")),
			common.BytesToHash([]byte("baz")),
		}
		key.MessageHeight = 30
		err = cache.Put(key, want)
		if err != nil {
			t.Fatal(err)
		}
		items, err := cache.Get(key, 3)
		if err != nil {
			t.Fatal(err)
		}
		if len(items) != len(want) {
			t.Fatalf("Wrong number of hashes. Expected %d, got %d", len(want), len(items))
		}
		if err = cache.Prune(ctx, 20); err != nil {
			t.Error(err)
		}
		items, err = cache.Get(key, 3)
		if err != nil {
			t.Fatal(err)
		}
		if len(items) != len(want) {
			t.Fatalf("Wrong number of hashes. Expected %d, got %d", len(want), len(items))
		}
	})
	t.Run("prunes many items with message number <= N", func(t *testing.T) {
		moduleRoots := []common.Hash{
			common.BytesToHash([]byte("foo")),
			common.BytesToHash([]byte("bar")),
			common.BytesToHash([]byte("baz")),
		}
		totalMessages := 10
		for _, root := range moduleRoots {
			for i := 0; i < totalMessages; i++ {
				hashes := []common.Hash{
					common.BytesToHash([]byte("a")),
					common.BytesToHash([]byte("b")),
					common.BytesToHash([]byte("c")),
				}
				key = &Key{
					WavmModuleRoot: root,
					MessageHeight:  uint64(i),
					StepHeights:    []uint64{0},
				}
				if err = cache.Put(key, hashes); err != nil {
					t.Fatal(err)
				}
			}
		}
		if err = cache.Prune(ctx, 5); err != nil {
			t.Error(err)
		}
		for _, root := range moduleRoots {
			// Expect that we deleted all entries with message number <= 5
			for i := 0; i <= 5; i++ {
				key = &Key{
					WavmModuleRoot: root,
					MessageHeight:  uint64(i),
					StepHeights:    []uint64{0},
				}
				if _, err = cache.Get(key, 3); !errors.Is(err, ErrNotFoundInCache) {
					t.Error(err)
				}
			}
			// But also expect that we kept all entries with message number > 5
			for i := 6; i < totalMessages; i++ {
				key = &Key{
					WavmModuleRoot: root,
					MessageHeight:  uint64(i),
					StepHeights:    []uint64{0},
				}
				items, err := cache.Get(key, 3)
				if err != nil {
					t.Error(err)
				}
				if len(items) != 3 {
					t.Fatalf("Wrong number of hashes. Expected %d, got %d", 3, len(items))
				}
			}
		}
	})
}

func TestReadWriteStatehashes(t *testing.T) {
	t.Run("read up to, but had empty reader", func(t *testing.T) {
		b := bytes.NewBuffer([]byte{})
		_, err := readHashes(b, 100)
		if err == nil {
			t.Fatal("Wanted error")
		}
		if !strings.Contains(err.Error(), "only read 0 hashes") {
			t.Fatal("Unexpected error")
		}
	})
	t.Run("read single root", func(t *testing.T) {
		b := bytes.NewBuffer([]byte{})
		want := common.BytesToHash([]byte("foo"))
		b.Write(want.Bytes())
		hashes, err := readHashes(b, 1)
		if err != nil {
			t.Fatal(err)
		}
		if len(hashes) == 0 {
			t.Fatal("Got no hashes")
		}
		if hashes[0] != want {
			t.Fatalf("Wrong root. Expected %#x, got %#x", want, hashes[0])
		}
	})
	t.Run("Three hashes exist, want to read only two", func(t *testing.T) {
		b := bytes.NewBuffer([]byte{})
		foo := common.BytesToHash([]byte("foo"))
		bar := common.BytesToHash([]byte("bar"))
		baz := common.BytesToHash([]byte("baz"))
		b.Write(foo.Bytes())
		b.Write(bar.Bytes())
		b.Write(baz.Bytes())
		hashes, err := readHashes(b, 2)
		if err != nil {
			t.Fatal(err)
		}
		if len(hashes) != 2 {
			t.Fatalf("Expected two hashes, got %d", len(hashes))
		}
		if hashes[0] != foo {
			t.Fatalf("Wrong root. Expected %#x, got %#x", foo, hashes[0])
		}
		if hashes[1] != bar {
			t.Fatalf("Wrong root. Expected %#x, got %#x", bar, hashes[1])
		}
	})
	t.Run("Fails to write enough data to writer", func(t *testing.T) {
		m := &mockWriter{wantErr: true}
		err := writeHashes(m, []common.Hash{common.BytesToHash([]byte("foo"))})
		if err == nil {
			t.Fatal("Wanted error")
		}
		m = &mockWriter{wantErr: false, numWritten: 16}
		err = writeHashes(m, []common.Hash{common.BytesToHash([]byte("foo"))})
		if err == nil {
			t.Fatal("Wanted error")
		}
		if !strings.Contains(err.Error(), "short write") {
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
	hashes    []common.Hash
	readIdx   int
	bytesRead int
}

func (m *mockReader) Read(out []byte) (n int, err error) {
	if m.wantErr {
		return 0, m.err
	}
	if m.readIdx == len(m.hashes) {
		return 0, io.EOF
	}
	copy(out, m.hashes[m.readIdx].Bytes())
	m.readIdx++
	return m.bytesRead, nil
}

func Test_readHashes(t *testing.T) {
	t.Run("Unexpected error", func(t *testing.T) {
		want := []common.Hash{
			common.BytesToHash([]byte("foo")),
			common.BytesToHash([]byte("bar")),
			common.BytesToHash([]byte("baz")),
		}
		m := &mockReader{wantErr: true, hashes: want, err: errors.New("foo")}
		_, err := readHashes(m, 1)
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
		m := &mockReader{wantErr: true, hashes: want, err: io.EOF}
		_, err := readHashes(m, 100)
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
		m := &mockReader{wantErr: false, hashes: want, bytesRead: 16}
		_, err := readHashes(m, 2)
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
		m := &mockReader{wantErr: false, hashes: want, bytesRead: 32}
		got, err := readHashes(m, 3)
		if err != nil {
			t.Fatal(err)
		}
		if len(want) != len(got) {
			t.Fatal("Wrong number of hashes")
		}
		for i, rt := range got {
			if rt != want[i] {
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
					StepHeights:   []uint64{50},
				},
			},
			want:    "wavm-module-root-0x0000000000000000000000000000000000000000000000000000000000000000/message-num-100-rollup-block-hash-0x0000000000000000000000000000000000000000000000000000000000000000/subchallenge-level-1-big-step-50/hashes.bin",
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
	ctx := context.Background()
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
	cache, err := New(basePath)
	if err != nil {
		b.Fatal(err)
	}
	if err = cache.Init(ctx); err != nil {
		b.Fatal(err)
	}
	key := &Key{
		WavmModuleRoot: common.BytesToHash([]byte("foo")),
		MessageHeight:  0,
		StepHeights:    []uint64{0},
	}
	numHashes := 1 << 20
	hashes := make([]common.Hash, numHashes)
	for i := range hashes {
		hashes[i] = common.BytesToHash([]byte(fmt.Sprintf("%d", i)))
	}
	if err := cache.Put(key, hashes); err != nil {
		b.Fatal(err)
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		readUpTo := uint64(1 << 20)
		hashes, err := cache.Get(key, readUpTo)
		if err != nil {
			b.Fatal(err)
		}
		if len(hashes) != numHashes {
			b.Fatalf("Wrong number of hashes. Expected %d, got %d", hashes, len(hashes))
		}
	}
}
