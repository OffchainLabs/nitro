// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package staker

import (
	"context"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/solgen/go/rollupgen"
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

type ValidatorWalletInterface interface {
	Initialize(context.Context) error
	Address() *common.Address
	AddressOrZero() common.Address
	TxSenderAddress() *common.Address
	RollupAddress() common.Address
	ChallengeManagerAddress() common.Address
	L1Client() arbutil.L1Interface
	TestTransactions(context.Context, []*types.Transaction) error
	ExecuteTransactions(context.Context, *ValidatorTxBuilder, common.Address) (*types.Transaction, error)
	TimeoutChallenges(context.Context, []uint64) (*types.Transaction, error)
	CanBatchTxs() bool
	AuthIfEoa() *bind.TransactOpts
}

type ContractValidatorWallet struct {
	con                     *rollupgen.ValidatorWallet
	address                 *common.Address
	onWalletCreated         func(common.Address)
	l1Reader                L1ReaderInterface
	auth                    *bind.TransactOpts
	walletFactoryAddr       common.Address
	rollupFromBlock         int64
	rollup                  *rollupgen.RollupUserLogic
	rollupAddress           common.Address
	challengeManagerAddress common.Address
}

var _ ValidatorWalletInterface = (*ContractValidatorWallet)(nil)

func NewContractValidatorWallet(address *common.Address, walletFactoryAddr, rollupAddress common.Address, l1Reader L1ReaderInterface, auth *bind.TransactOpts, rollupFromBlock int64, onWalletCreated func(common.Address)) (*ContractValidatorWallet, error) {
	var con *rollupgen.ValidatorWallet
	if address != nil {
		var err error
		con, err = rollupgen.NewValidatorWallet(*address, l1Reader.Client())
		if err != nil {
			return nil, err
		}
	}
	rollup, err := rollupgen.NewRollupUserLogic(rollupAddress, l1Reader.Client())
	if err != nil {
		return nil, err
	}
	return &ContractValidatorWallet{
		con:               con,
		address:           address,
		onWalletCreated:   onWalletCreated,
		l1Reader:          l1Reader,
		auth:              auth,
		walletFactoryAddr: walletFactoryAddr,
		rollupAddress:     rollupAddress,
		rollup:            rollup,
		rollupFromBlock:   rollupFromBlock,
	}, nil
}

func (v *ContractValidatorWallet) validateWallet(ctx context.Context) error {
	if v.con == nil || v.auth == nil {
		return nil
	}
	callOpts := &bind.CallOpts{Context: ctx}
	owner, err := v.con.Owner(callOpts)
	if err != nil {
		return err
	}
	isExecutor, err := v.con.Executors(callOpts, v.auth.From)
	if err != nil {
		return err
	}
	if v.auth.From != owner && !isExecutor {
		return errors.New("specified unauthorized smart contract wallet")
	}
	return nil
}

func (v *ContractValidatorWallet) Initialize(ctx context.Context) error {
	err := v.populateWallet(ctx, false)
	if err != nil {
		return err
	}
	err = v.validateWallet(ctx)
	if err != nil {
		return err
	}
	callOpts := &bind.CallOpts{Context: ctx}
	v.challengeManagerAddress, err = v.rollup.ChallengeManager(callOpts)
	return err
}

// May be the nil if the wallet hasn't been deployed yet
func (v *ContractValidatorWallet) Address() *common.Address {
	return v.address
}

// May be zero if the wallet hasn't been deployed yet
func (v *ContractValidatorWallet) AddressOrZero() common.Address {
	if v.address == nil {
		return common.Address{}
	}
	return *v.address
}

func (v *ContractValidatorWallet) TxSenderAddress() *common.Address {
	if v.auth == nil {
		return nil
	}
	return &v.auth.From
}

func (v *ContractValidatorWallet) From() common.Address {
	if v.auth == nil {
		return common.Address{}
	}
	return v.auth.From
}

func (v *ContractValidatorWallet) executeTransaction(ctx context.Context, tx *types.Transaction, gasRefunder common.Address) (*types.Transaction, error) {
	oldAuthValue := v.auth.Value
	v.auth.Value = tx.Value()
	defer (func() { v.auth.Value = oldAuthValue })()

	return v.con.ExecuteTransactionWithGasRefunder(v.auth, gasRefunder, tx.Data(), *tx.To(), tx.Value())
}

func (v *ContractValidatorWallet) populateWallet(ctx context.Context, createIfMissing bool) error {
	if v.con != nil {
		return nil
	}
	if v.auth == nil {
		if createIfMissing {
			return errors.New("cannot create validator smart contract wallet without key wallet")
		}
		return nil
	}
	if v.address == nil {
		addr, err := GetValidatorWalletContract(ctx, v.walletFactoryAddr, v.rollupFromBlock, v.auth, v.l1Reader, createIfMissing)
		if err != nil {
			return err
		}
		if addr == nil {
			return nil
		}
		v.address = addr
		if v.onWalletCreated != nil {
			v.onWalletCreated(*addr)
		}
	}
	con, err := rollupgen.NewValidatorWallet(*v.address, v.l1Reader.Client())
	if err != nil {
		return err
	}
	v.con = con

	if err := v.validateWallet(ctx); err != nil {
		return err
	}
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
func (v *ContractValidatorWallet) ExecuteTransactions(ctx context.Context, builder *ValidatorTxBuilder, gasRefunder common.Address) (*types.Transaction, error) {
	txes := builder.transactions
	if len(txes) == 0 {
		return nil, nil
	}

	err := v.populateWallet(ctx, true)
	if err != nil {
		return nil, err
	}

	if len(txes) == 1 {
		arbTx, err := v.executeTransaction(ctx, txes[0], gasRefunder)
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

	balanceInContract, err := v.l1Reader.Client().BalanceAt(ctx, *v.address, nil)
	if err != nil {
		return nil, err
	}

	oldAuthValue := v.auth.Value
	v.auth.Value = new(big.Int).Sub(totalAmount, balanceInContract)
	if v.auth.Value.Sign() < 0 {
		v.auth.Value.SetInt64(0)
	}
	defer (func() { v.auth.Value = oldAuthValue })()

	arbTx, err := v.con.ExecuteTransactionsWithGasRefunder(v.auth, gasRefunder, data, dest, amount)
	if err != nil {
		return nil, err
	}
	builder.transactions = nil
	return arbTx, nil
}

func (v *ContractValidatorWallet) TimeoutChallenges(ctx context.Context, challenges []uint64) (*types.Transaction, error) {
	return v.con.TimeoutChallenges(v.auth, v.challengeManagerAddress, challenges)
}

func (v *ContractValidatorWallet) L1Client() arbutil.L1Interface {
	return v.l1Reader.Client()
}

func (v *ContractValidatorWallet) RollupAddress() common.Address {
	return v.rollupAddress
}

func (v *ContractValidatorWallet) ChallengeManagerAddress() common.Address {
	return v.challengeManagerAddress
}

func (v *ContractValidatorWallet) TestTransactions(ctx context.Context, txs []*types.Transaction) error {
	if v.Address() == nil {
		return nil
	}
	data, dest, amount, totalAmount := combineTxes(txs)
	realData, err := validatorABI.Pack("executeTransactions", data, dest, amount)
	if err != nil {
		return err
	}
	msg := ethereum.CallMsg{
		From:  v.From(),
		To:    v.Address(),
		Value: totalAmount,
		Data:  realData,
	}
	_, err = v.L1Client().PendingCallContract(ctx, msg)
	return err
}

func (v *ContractValidatorWallet) CanBatchTxs() bool {
	return true
}

func (v *ContractValidatorWallet) AuthIfEoa() *bind.TransactOpts {
	return nil
}

func GetValidatorWalletContract(
	ctx context.Context,
	validatorWalletFactoryAddr common.Address,
	fromBlock int64,
	transactAuth *bind.TransactOpts,
	l1Reader L1ReaderInterface,
	createIfMissing bool,
) (*common.Address, error) {
	client := l1Reader.Client()

	// TODO: If we just save a mapping in the wallet creator we won't need log search
	walletCreator, err := rollupgen.NewValidatorWalletCreator(validatorWalletFactoryAddr, client)
	if err != nil {
		return nil, errors.WithStack(err)
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
		return nil, errors.WithStack(err)
	}
	if len(logs) > 1 {
		return nil, errors.New("more than one validator wallet created for address")
	}
	if len(logs) == 1 {
		rawLog := logs[0]
		parsed, err := walletCreator.ParseWalletCreated(rawLog)
		if err != nil {
			return nil, err
		}
		log.Info("found validator smart contract wallet", "address", parsed.WalletAddress)
		return &parsed.WalletAddress, err
	}

	if !createIfMissing {
		return nil, nil
	}

	var initialExecutorAllowedDests []common.Address
	tx, err := walletCreator.CreateWallet(transactAuth, initialExecutorAllowedDests)
	if err != nil {
		return nil, err
	}

	receipt, err := l1Reader.WaitForTxApproval(ctx, tx)
	if err != nil {
		return nil, err
	}
	ev, err := walletCreator.ParseWalletCreated(*receipt.Logs[len(receipt.Logs)-1])
	if err != nil {
		return nil, errors.WithStack(err)
	}
	log.Info("created validator smart contract wallet", "address", ev.WalletAddress)
	return &ev.WalletAddress, nil
}
