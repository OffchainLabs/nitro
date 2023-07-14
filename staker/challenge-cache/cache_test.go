package challengecache

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	protocol "github.com/OffchainLabs/challenge-protocol-v2/chain-abstraction"
	"github.com/OffchainLabs/challenge-protocol-v2/containers/option"
	"github.com/ethereum/go-ethereum/common"
)

var _ HistoryCommitmentCacher = (*Cache)(nil)

func TestCache(t *testing.T) {
	basePath, err := ioutil.TempDir("", "*")
	if err != nil {
		t.Fatal(err)
	}
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
		AssertionHash:  common.BytesToHash([]byte("bar")),
		MessageRange:   HeightRange{From: 0, To: 1},
		BigStepRange: option.Some(HeightRange{
			From: 0, To: 1,
		}),
		ToSmallStep: option.Some(protocol.Height(100)),
	}
	want := []common.Hash{
		common.BytesToHash([]byte("foo")),
		common.BytesToHash([]byte("bar")),
		common.BytesToHash([]byte("baz")),
	}
	err = cache.Put(key, want)
	if err != nil {
		t.Fatal(err)
	}
	got, err := cache.Get(key, option.None[protocol.Height]())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(want) {
		t.Fatalf("Wrong number of roots. Expected %d, got %d", len(want), len(got))
	}
	for i, rt := range got {
		if rt != want[i] {
			t.Fatalf("Wrong root. Expected %#x, got %#x", want[i], rt)
		}
	}
}

func TestReadWriteStateRoots(t *testing.T) {
	t.Run("read empty", func(t *testing.T) {
		b := bytes.NewBuffer([]byte{})
		roots, err := readStateRoots(b, option.None[protocol.Height]())
		if err != nil {
			t.Fatal(err)
		}
		if len(roots) != 0 {
			t.Fatal("Expected no roots")
		}
	})
	t.Run("read up to, but had empty reader", func(t *testing.T) {
		b := bytes.NewBuffer([]byte{})
		_, err := readStateRoots(b, option.Some(protocol.Height(100)))
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
		roots, err := readStateRoots(b, option.Some(protocol.Height(0)))
		if err != nil {
			t.Fatal(err)
		}
		if len(roots) == 0 {
			t.Fatal("Got no roots")
		}
		if roots[0] != want {
			t.Fatalf("Wrong root. Expected %#x, got %#x", want, roots[0])
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
		roots, err := readStateRoots(b, option.Some(protocol.Height(1)))
		if err != nil {
			t.Fatal(err)
		}
		if len(roots) != 2 {
			t.Fatalf("Expected two roots, got %d", len(roots))
		}
		if roots[0] != foo {
			t.Fatalf("Wrong root. Expected %#x, got %#x", foo, roots[0])
		}
		if roots[1] != bar {
			t.Fatalf("Wrong root. Expected %#x, got %#x", bar, roots[1])
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
			name: "bad message range",
			args: args{
				baseDir: "",
				key: &Key{
					MessageRange: HeightRange{
						From: 1, To: 0,
					},
				},
			},
			wantErr:     true,
			errContains: "message number range invalid",
		},
		{
			name: "bad message range equal",
			args: args{
				baseDir: "",
				key: &Key{
					MessageRange: HeightRange{
						From: 100, To: 100,
					},
				},
			},
			wantErr:     true,
			errContains: "message number range invalid",
		},
		{
			name: "message range not at one step fork",
			args: args{
				baseDir: "",
				key: &Key{
					MessageRange: HeightRange{
						From: 100, To: 102,
					},
					BigStepRange: option.Some(HeightRange{
						From: 0, To: 1,
					}),
				},
			},
			wantErr:     true,
			errContains: "message number range invalid",
		},
		{
			name: "big step range invalid",
			args: args{
				baseDir: "",
				key: &Key{
					MessageRange: HeightRange{
						From: 100, To: 101,
					},
					BigStepRange: option.Some(HeightRange{
						From: 1, To: 0,
					}),
				},
			},
			wantErr:     true,
			errContains: "big step range invalid",
		},
		{
			name: "big step range not at one step fork",
			args: args{
				baseDir: "",
				key: &Key{
					MessageRange: HeightRange{
						From: 100, To: 101,
					},
					BigStepRange: option.Some(HeightRange{
						From: 100, To: 102,
					}),
					ToSmallStep: option.Some(protocol.Height(100)),
				},
			},
			wantErr:     true,
			errContains: "big step range invalid",
		},
		{
			name: "OK",
			args: args{
				baseDir: "",
				key: &Key{
					MessageRange: HeightRange{
						From: 100, To: 101,
					},
					BigStepRange: option.Some(HeightRange{
						From: 50, To: 51,
					}),
					ToSmallStep: option.Some(protocol.Height(100)),
				},
			},
			want:    "wavm-module-root-0x0000000000000000000000000000000000000000000000000000000000000000/assertion-0x0000000000000000000000000000000000000000000000000000000000000000/message-num-100-101/big-step-50-51/small-step-0-100/roots.txt",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := determineFilePath(tt.args.baseDir, tt.args.key)
			if (err != nil) != tt.wantErr {
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
	basePath, err := ioutil.TempDir("", "*")
	if err != nil {
		b.Fatal(err)
	}
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
		AssertionHash:  common.BytesToHash([]byte("bar")),
		MessageRange:   HeightRange{From: 0, To: 1},
		BigStepRange: option.Some(HeightRange{
			From: 0, To: 1,
		}),
		ToSmallStep: option.Some(protocol.Height(100)),
	}
	numRoots := 1 << 20
	roots := make([]common.Hash, numRoots)
	for i := range roots {
		roots[i] = common.BytesToHash([]byte(fmt.Sprintf("%d", i)))
	}
	if err = cache.Put(key, roots); err != nil {
		b.Fatal(err)
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		readUpTo := option.None[protocol.Height]()
		roots, err := cache.Get(key, readUpTo)
		if err != nil {
			b.Fatal(err)
		}
		if len(roots) != numRoots {
			b.Fatalf("Wrong number of roots. Expected %d, got %d", numRoots, len(roots))
		}
	}
}
