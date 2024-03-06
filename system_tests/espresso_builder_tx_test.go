package arbtest

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/big"
	"testing"

	espressoTypes "github.com/EspressoSystems/espresso-sequencer-go/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbos/l2pricing"
	"github.com/offchainlabs/nitro/statetransfer"
)

func TestEspressoBuilderGuaranteedTx(t *testing.T) {
	l1Header := arbostypes.L1IncomingMessageHeader{
		Kind:      arbostypes.L1MessageType_L2Message,
		L1BaseFee: big.NewInt(10000),
		Poster:    common.Address{},
	}
	chainDb := rawdb.NewMemoryDatabase()
	chainConfig := params.ArbitrumDevTestChainConfig()
	chainConfig.ArbitrumChainParams.EnableEspresso = true

	serializedChainConfig, err := json.Marshal(chainConfig)
	if err != nil {
		panic(err)
	}
	initMessage := &arbostypes.ParsedInitMessage{
		ChainId:               chainConfig.ChainID,
		InitialL1BaseFee:      arbostypes.DefaultInitialL1BaseFee,
		ChainConfig:           chainConfig,
		SerializedChainConfig: serializedChainConfig,
	}
	stateRoot, err := arbosState.InitializeArbosInDatabase(
		chainDb,
		statetransfer.NewMemoryInitDataReader(&statetransfer.ArbosInitializationInfo{}),
		chainConfig,
		initMessage,
		0,
		0,
	)
	if err != nil {
		panic(err)
	}
	statedb, err := state.New(stateRoot, state.NewDatabase(chainDb), nil)
	if err != nil {
		panic(err)
	}

	chainContext := noopChainContext{}
	seqBatch := make([]byte, 40)
	binary.BigEndian.PutUint64(seqBatch[8:16], ^uint64(0))
	binary.BigEndian.PutUint64(seqBatch[24:32], ^uint64(0))
	binary.BigEndian.PutUint64(seqBatch[32:40], uint64(0))
	genesis := &types.Header{
		Number:     new(big.Int),
		Nonce:      types.EncodeNonce(0),
		Time:       0,
		ParentHash: common.Hash{},
		Extra:      []byte("Arbitrum"),
		GasLimit:   l2pricing.GethBlockGasLimit,
		GasUsed:    0,
		BaseFee:    big.NewInt(l2pricing.InitialBaseFeeWei),
		Difficulty: big.NewInt(1),
		MixDigest:  common.Hash{},
		Coinbase:   common.Address{},
		Root:       stateRoot,
	}

	builderAddr := common.HexToAddress("0xa3612e81E1f9cdF8f54C3d65f7FBc0aBf5B21E8f")
	// Since we are not really building a block, so this illegal header is enough
	espressoHeader := espressoTypes.Header{
		FeeInfo: &espressoTypes.FeeInfo{Account: builderAddr},
		Height:  10,
	}

	testInfo := NewArbTestInfo(t, chainConfig.ChainID)

	otherBuilderTx := testInfo.PrepareTx("Owner", "Faucet", 1000000, big.NewInt(1000000), getExtraBytes(common.Address{10: 5}))
	hooks := arbos.NoopSequencingHooks()
	_, _, err = arbos.ProduceBlockAdvanced(&l1Header, types.Transactions{otherBuilderTx}, 0, genesis, statedb, chainContext, chainConfig, hooks, &espressoHeader)
	if err != nil {
		panic(err)
	}
	txErr := hooks.TxErrors[0]
	if txErr == nil {
		panic("this tx should not be executed successfully")
	}
	if txErr.Error() != arbos.NOT_EXPECTED_BUILDER_ERROR {
		panic(fmt.Sprintf("error should be %s", arbos.NOT_EXPECTED_BUILDER_ERROR))
	}

	hooks2 := arbos.NoopSequencingHooks()
	thisBuilderTx := testInfo.PrepareTx("Owner", "Faucet", 1000000, big.NewInt(1000000), getExtraBytes(builderAddr))
	_, _, err = arbos.ProduceBlockAdvanced(&l1Header, types.Transactions{thisBuilderTx}, 0, genesis, statedb, chainContext, chainConfig, hooks2, &espressoHeader)
	if err != nil {
		panic(err)
	}
	txErr2 := hooks2.TxErrors[0]
	if txErr2 == nil {
		panic("this tx should not be executed successfully")
	}
	if txErr2.Error() == arbos.NOT_EXPECTED_BUILDER_ERROR {
		panic(fmt.Sprintf("error should not be %s", arbos.NOT_EXPECTED_BUILDER_ERROR))
	}
}

func getExtraBytes(addr common.Address) []byte {
	result := [52]byte{}
	magicBytes := espressoTypes.GetMagicBytes()
	copy(result[0:32], magicBytes[:])
	copy(result[32:], addr.Bytes())
	return result[:]
}
