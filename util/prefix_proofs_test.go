package util_test

import (
	"context"
	"testing"

	"fmt"

	statemanager "github.com/OffchainLabs/challenge-protocol-v2/state-manager"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func TestGoPrefixProofs(t *testing.T) {
	ctx := context.Background()
	hashes := make([]common.Hash, 10)
	for i := 0; i < len(hashes); i++ {
		hashes[i] = crypto.Keccak256Hash([]byte(fmt.Sprintf("%d", i)))
	}
	manager := statemanager.New(hashes)

	loCommit, err := manager.HistoryCommitmentUpTo(ctx, 3)
	require.NoError(t, err)
	hiCommit, err := manager.HistoryCommitmentUpTo(ctx, 7)
	require.NoError(t, err)
	packedProof, err := manager.PrefixProof(ctx, 3, 7)
	require.NoError(t, err)

	data, err := statemanager.ProofArgs.Unpack(packedProof)
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

	err = util.VerifyPrefixProofGo(&util.VerifyPrefixProofConfig{
		PreRoot:      loCommit.Merkle,
		PreSize:      4,
		PostRoot:     hiCommit.Merkle,
		PostSize:     8,
		PreExpansion: preExpansionHashes,
		PrefixProof:  prefixProof,
	})
	require.NoError(t, err)
}
