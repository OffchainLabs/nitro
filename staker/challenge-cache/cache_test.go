package challengecache

import (
	"os"
	"testing"

	protocol "github.com/OffchainLabs/challenge-protocol-v2/chain-abstraction"
	"github.com/OffchainLabs/challenge-protocol-v2/containers/option"
	"github.com/ethereum/go-ethereum/common"
)

func TestCachePut(t *testing.T) {
	basePath := "/tmp/testingcache"
	t.Cleanup(func() {
		if err := os.RemoveAll(basePath); err != nil {
			t.Fatal(err)
		}
	})
	roots := []common.Hash{}
	cache, err := New(basePath)
	if err != nil {
		t.Fatal(err)
	}
	err = cache.Put(&Key{
		WavmModuleRoot: common.BytesToHash([]byte("foo")),
		AssertionHash:  common.BytesToHash([]byte("bar")),
		MessageRange:   HeightRange{from: 0, to: 1},
		BigStepRange: option.Some(HeightRange{
			from: 0, to: 1,
		}),
		ToSmallStep: option.Some(protocol.Height(100)),
	}, roots)
	if err != nil {
		t.Fatal(err)
	}
}

func BenchmarkCacheRead(b *testing.B) {

}
