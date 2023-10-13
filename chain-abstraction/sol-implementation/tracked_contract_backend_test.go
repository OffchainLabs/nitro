package solimpl

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestTrackedContractBackend(t *testing.T) {
	backend := NewTrackedContractBackend(&MockContractBackend{})

	t.Run("CallContract", func(t *testing.T) {
		data := []byte("1234mockdata1283918231923")
		call := ethereum.CallMsg{Data: data}
		_, err := backend.CallContract(context.Background(), call, nil)
		require.NoError(t, err)

		key := fmt.Sprintf("%#x", data[:4])
		metric, ok := backend.metrics[key]
		require.True(t, ok)
		require.Equal(t, 1, metric.Calls)
	})

	t.Run("SendTransaction", func(t *testing.T) {
		data := []byte("1234mocktx1283918231923")
		tx := types.NewTransaction(1, common.HexToAddress("0x123456789"), big.NewInt(1), 21000, big.NewInt(1), data)
		err := backend.SendTransaction(context.Background(), tx)
		require.NoError(t, err)

		key := fmt.Sprintf("%#x", data[:4])
		metric, ok := backend.metrics[key]
		require.True(t, ok)
		require.Equal(t, 1, metric.Txs)
		require.Equal(t, big.NewInt(21000), &metric.GasCosts[0])
	})
}

func Test_median(t *testing.T) {
	t.Run("Empty slice", func(t *testing.T) {
		var gasCosts []big.Int
		med := median(gasCosts)
		require.Equal(t, true, med == nil)
	})

	t.Run("Single value", func(t *testing.T) {
		gasCosts := []big.Int{*big.NewInt(5)}
		med := median(gasCosts)
		require.Equal(t, big.NewInt(5), med)
	})

	t.Run("Two values takes mean", func(t *testing.T) {
		gasCosts := []big.Int{*big.NewInt(5), *big.NewInt(15)}
		med := median(gasCosts)
		require.Equal(t, big.NewInt(7), med)
	})

	t.Run("Odd number of values", func(t *testing.T) {
		gasCosts := []big.Int{*big.NewInt(5), *big.NewInt(15), *big.NewInt(25)}
		med := median(gasCosts)
		require.Equal(t, big.NewInt(15), med)
	})

	t.Run("Unsorted values", func(t *testing.T) {
		gasCosts := []big.Int{*big.NewInt(25), *big.NewInt(5), *big.NewInt(15)}
		med := median(gasCosts)
		require.Equal(t, big.NewInt(15), med)
	})
}

type MockContractBackend struct{}

func (m *MockContractBackend) CodeAt(ctx context.Context, contract common.Address, blockNumber *big.Int) ([]byte, error) {
	return nil, nil
}

func (m *MockContractBackend) CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	return nil, nil
}

func (m *MockContractBackend) PendingCodeAt(ctx context.Context, contract common.Address) ([]byte, error) {
	return nil, nil
}

func (m *MockContractBackend) PendingCallContract(ctx context.Context, call ethereum.CallMsg) ([]byte, error) {
	return nil, nil
}

func (m *MockContractBackend) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	return nil, nil
}

func (m *MockContractBackend) PendingNonceAt(ctx context.Context, account common.Address) (uint64, error) {
	return 0, nil
}

func (m *MockContractBackend) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	return big.NewInt(1), nil
}

func (m *MockContractBackend) SuggestGasTipCap(ctx context.Context) (*big.Int, error) {
	return big.NewInt(1), nil
}

func (m *MockContractBackend) EstimateGas(ctx context.Context, call ethereum.CallMsg) (gas uint64, err error) {
	return 0, nil
}

func (m *MockContractBackend) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	return nil
}

func (m *MockContractBackend) FilterLogs(ctx context.Context, query ethereum.FilterQuery) ([]types.Log, error) {
	return nil, nil
}

func (m *MockContractBackend) SubscribeFilterLogs(ctx context.Context, query ethereum.FilterQuery, ch chan<- types.Log) (ethereum.Subscription, error) {
	return nil, nil
}

func (m *MockContractBackend) TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	return nil, nil
}
