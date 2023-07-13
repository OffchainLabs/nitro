package challengecache

import (
	"fmt"
	"os"
	"testing"

	protocol "github.com/OffchainLabs/challenge-protocol-v2/chain-abstraction"
	"github.com/OffchainLabs/challenge-protocol-v2/containers/option"
	"github.com/ethereum/go-ethereum/common"
)

func TestCache(t *testing.T) {
	basePath := "/tmp/testingcache"
	t.Cleanup(func() {
		if err := os.RemoveAll(basePath); err != nil {
			t.Fatal(err)
		}
	})
	cache, err := New(basePath)
	if err != nil {
		t.Fatal(err)
	}
	key := &Key{
		WavmModuleRoot: common.BytesToHash([]byte("foo")),
		AssertionHash:  common.BytesToHash([]byte("bar")),
		MessageRange:   HeightRange{from: 0, to: 1},
		BigStepRange: option.Some(HeightRange{
			from: 0, to: 1,
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

func BenchmarkCache_Read_32Mb(b *testing.B) {
	b.StopTimer()
	basePath := "/tmp/testingcache"
	b.Cleanup(func() {
		if err := os.RemoveAll(basePath); err != nil {
			b.Fatal(err)
		}
	})
	cache, err := New(basePath)
	if err != nil {
		b.Fatal(err)
	}
	key := &Key{
		WavmModuleRoot: common.BytesToHash([]byte("foo")),
		AssertionHash:  common.BytesToHash([]byte("bar")),
		MessageRange:   HeightRange{from: 0, to: 1},
		BigStepRange: option.Some(HeightRange{
			from: 0, to: 1,
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
