//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package validator

import (
	"context"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/offchainlabs/arbstate/arbutil"
	"github.com/offchainlabs/arbstate/solgen/go/rollupgen"
	"github.com/pkg/errors"
)

var validatorABI abi.ABI
var walletCreatedID common.Hash

func init() {
	parsedValidator, err := abi.JSON(strings.NewReader(rollupgen.ValidatorWalletABI))
	if err != nil {
		panic(err)
	}
	validatorABI = parsedValidator

	parsedValidatorWalletCreator, err := abi.JSON(strings.NewReader(rollupgen.ValidatorWalletCreatorABI))
	if err != nil {
		panic(err)
	}
	walletCreatedID = parsedValidatorWalletCreator.Events["WalletCreated"].ID
}

type ValidatorWallet struct {
	con               *rollupgen.ValidatorWallet
	address           *common.Address
	onWalletCreated   func(common.Address)
	client            arbutil.L1Interface
	auth              *bind.TransactOpts
	rollupAddress     common.Address
	walletFactoryAddr common.Address
	rollupFromBlock   int64
}

func NewValidatorWallet(address *common.Address, walletFactoryAddr, rollupAddress common.Address, client arbutil.L1Interface, auth *bind.TransactOpts, rollupFromBlock int64, onWalletCreated func(common.Address)) (*ValidatorWallet, error) {
	var con *rollupgen.ValidatorWallet
	if address != nil {
		var err error
		con, err = rollupgen.NewValidatorWallet(*address, client)
		if err != nil {
			return nil, err
		}
	}
	return &ValidatorWallet{
		con:               con,
		address:           address,
		onWalletCreated:   onWalletCreated,
		client:            client,
		auth:              auth,
		rollupAddress:     rollupAddress,
		walletFactoryAddr: walletFactoryAddr,
		rollupFromBlock:   rollupFromBlock,
	}, nil
}

// May be the nil if the wallet hasn't been deployed yet
func (v *ValidatorWallet) Address() *common.Address {
	return v.address
}

func (v *ValidatorWallet) From() common.Address {
	return v.auth.From
}

func (v *ValidatorWallet) RollupAddress() common.Address {
	return v.rollupAddress
}

func (v *ValidatorWallet) executeTransaction(ctx context.Context, tx *types.Transaction) (*types.Transaction, error) {
	oldAuthValue := v.auth.Value
	v.auth.Value = tx.Value()
	defer (func() { v.auth.Value = oldAuthValue })()

	return v.con.ExecuteTransaction(v.auth, tx.Data(), *tx.To(), tx.Value())
}

func (v *ValidatorWallet) createWalletIfNeeded(ctx context.Context) error {
	if v.con != nil {
		return nil
	}
	if v.address == nil {
		addr, err := CreateValidatorWallet(ctx, v.walletFactoryAddr, v.rollupFromBlock, v.auth, v.client)
		if err != nil {
			return err
		}
		v.address = &addr
		if v.onWalletCreated != nil {
			v.onWalletCreated(addr)
		}
	}
	con, err := rollupgen.NewValidatorWallet(*v.address, v.client)
	if err != nil {
		return err
	}
	v.con = con
	return nil
}

func combineTxes(txes []*types.Transaction) ([][]byte, []common.Address, []*big.Int, *big.Int) {
	totalAmount := big.NewInt(0)
	data := make([][]byte, 0, len(txes))
	dest := make([]common.Address, 0, len(txes))
	amount := make([]*big.Int, 0, len(txes))

	for _, tx := range txes {
		data = append(data, tx.Data())
		dest = append(dest, *tx.To())
		amount = append(amount, tx.Value())
		totalAmount = totalAmount.Add(totalAmount, tx.Value())
	}
	return data, dest, amount, totalAmount
}

// Not thread safe! Don't call this from multiple threads at the same time.
func (v *ValidatorWallet) ExecuteTransactions(ctx context.Context, builder *BuilderBackend) (*types.Transaction, error) {
	txes := builder.transactions
	if len(txes) == 0 {
		return nil, nil
	}

	err := v.createWalletIfNeeded(ctx)
	if err != nil {
		return nil, err
	}

	if len(txes) == 1 {
		arbTx, err := v.executeTransaction(ctx, txes[0])
		if err != nil {
			return nil, err
		}
		builder.transactions = nil
		return arbTx, nil
	}

	totalAmount := big.NewInt(0)
	data := make([][]byte, 0, len(txes))
	dest := make([]common.Address, 0, len(txes))
	amount := make([]*big.Int, 0, len(txes))

	for _, tx := range txes {
		data = append(data, tx.Data())
		dest = append(dest, *tx.To())
		amount = append(amount, tx.Value())
		totalAmount = totalAmount.Add(totalAmount, tx.Value())
	}

	oldAuthValue := v.auth.Value
	v.auth.Value = totalAmount
	defer (func() { v.auth.Value = oldAuthValue })()

	arbTx, err := v.con.ExecuteTransactions(v.auth, data, dest, amount)
	if err != nil {
		return nil, err
	}
	builder.transactions = nil
	return arbTx, nil
}

func (v *ValidatorWallet) ReturnOldDeposits(ctx context.Context, stakers []common.Address) (*types.Transaction, error) {
	return v.con.ReturnOldDeposits(v.auth, v.rollupAddress, stakers)
}

func (v *ValidatorWallet) TimeoutChallenges(ctx context.Context, challenges []common.Address) (*types.Transaction, error) {
	return v.con.TimeoutChallenges(v.auth, challenges)
}

func CreateValidatorWallet(
	ctx context.Context,
	validatorWalletFactoryAddr common.Address,
	fromBlock int64,
	transactAuth *bind.TransactOpts,
	client arbutil.L1Interface,
) (common.Address, error) {
	// TODO: If we just save a mapping in the wallet creator we won't need log search
	walletCreator, err := rollupgen.NewValidatorWalletCreator(validatorWalletFactoryAddr, client)
	if err != nil {
		return common.Address{}, errors.WithStack(err)
	}

	query := ethereum.FilterQuery{
		BlockHash: nil,
		FromBlock: big.NewInt(fromBlock),
		ToBlock:   nil,
		Addresses: []common.Address{validatorWalletFactoryAddr},
		Topics:    [][]common.Hash{{walletCreatedID}, nil, {transactAuth.From.Hash()}},
	}
	logs, err := client.FilterLogs(ctx, query)
	if err != nil {
		return common.Address{}, errors.WithStack(err)
	}
	if len(logs) > 1 {
		return common.Address{}, errors.New("more than one validator wallet created for address")
	} else if len(logs) == 1 {
		log := logs[0]
		parsed, err := walletCreator.ParseWalletCreated(log)
		if err != nil {
			return common.Address{}, err
		}
		return parsed.WalletAddress, err
	}

	tx, err := walletCreator.CreateWallet(transactAuth)
	if err != nil {
		return common.Address{}, err
	}

	receipt, err := arbutil.EnsureTxSucceededWithTimeout(
		ctx,
		client,
		tx,
		txTimeout,
	)
	if err != nil {
		return common.Address{}, err
	}
	ev, err := walletCreator.ParseWalletCreated(*receipt.Logs[len(receipt.Logs)-1])
	if err != nil {
		return common.Address{}, errors.WithStack(err)
	}
	return ev.WalletAddress, nil
}
