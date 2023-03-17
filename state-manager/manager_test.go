package statemanager

import (
	"context"
	"encoding/binary"
	"testing"

	"github.com/OffchainLabs/challenge-protocol-v2/util/prefix-proofs"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func TestPrefixProofs(t *testing.T) {
	ctx := context.Background()
	for _, c := range []struct {
		lo uint64
		hi uint64
	}{
		{0, 1},
		{0, 3},
		{1, 2},
		{1, 3},
		{1, 15},
		{17, 255},
		{23, 255},
		{20, 511},
	} {
		leaves := hashesForUints(0, c.hi+1)
		manager := New(leaves)
		packedProof, err := manager.PrefixProof(ctx, c.lo, c.hi)
		require.NoError(t, err)

		data, err := ProofArgs.Unpack(packedProof)
		require.NoError(t, err)
		preExpansion := data[0].([][32]byte)
		proof := data[1].([][32]byte)

		preExpansionHashes := make([]common.Hash, len(preExpansion))
		for i := 0; i < len(preExpansion); i++ {
			preExpansionHashes[i] = preExpansion[i]
		}
		prefixProof := make([]common.Hash, len(proof))
		for i := 0; i < len(proof); i++ {
			prefixProof[i] = proof[i]
		}

		postExpansion, err := manager.HistoryCommitmentUpTo(ctx, c.hi)
		require.NoError(t, err)

		cfg := &prefixproofs.VerifyPrefixProofConfig{
			PreRoot:      prefixproofs.Root(preExpansionHashes),
			PreSize:      c.lo + 1,
			PostRoot:     postExpansion.Merkle,
			PostSize:     c.hi + 1,
			PreExpansion: preExpansionHashes,
			PrefixProof:  prefixProof,
		}
		err = prefixproofs.VerifyPrefixProof(cfg)
		require.NoError(t, err)
	}
}

func hashesForUints(lo, hi uint64) []common.Hash {
	ret := []common.Hash{}
	for i := lo; i < hi; i++ {
		ret = append(ret, hashForUint(i))
	}
	return ret
}

func hashForUint(x uint64) common.Hash {
	return crypto.Keccak256Hash(binary.BigEndian.AppendUint64([]byte{}, x))
}
