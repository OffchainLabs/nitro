// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package staker

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbnode/dataposter"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/solgen/go/rollupgen"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/headerreader"
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
	// Address must be able to be called concurrently with other functions
	Address() *common.Address
	// Address must be able to be called concurrently with other functions
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
	Start(context.Context)
	StopAndWait()
	// May be nil
	DataPoster() *dataposter.DataPoster
}

type ContractValidatorWallet struct {
	con                     *rollupgen.ValidatorWallet
	address                 atomic.Pointer[common.Address]
	onWalletCreated         func(common.Address)
	l1Reader                *headerreader.HeaderReader
	auth                    *bind.TransactOpts
	walletFactoryAddr       common.Address
	rollupFromBlock         int64
	rollup                  *rollupgen.RollupUserLogic
	rollupAddress           common.Address
	challengeManagerAddress common.Address
	dataPoster              *dataposter.DataPoster
	getExtraGas             func() uint64
}

var _ ValidatorWalletInterface = (*ContractValidatorWallet)(nil)

func NewContractValidatorWallet(dp *dataposter.DataPoster, address *common.Address, walletFactoryAddr, rollupAddress common.Address, l1Reader *headerreader.HeaderReader, auth *bind.TransactOpts, rollupFromBlock int64, onWalletCreated func(common.Address),
	getExtraGas func() uint64) (*ContractValidatorWallet, error) {
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
	wallet := &ContractValidatorWallet{
		con:               con,
		onWalletCreated:   onWalletCreated,
		l1Reader:          l1Reader,
		auth:              auth,
		walletFactoryAddr: walletFactoryAddr,
		rollupAddress:     rollupAddress,
		rollup:            rollup,
		rollupFromBlock:   rollupFromBlock,
		dataPoster:        dp,
		getExtraGas:       getExtraGas,
	}
	// Go complains if we make an address variable before wallet and copy it in
	wallet.address.Store(address)
	return wallet, nil
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
	return v.address.Load()
}

// May be zero if the wallet hasn't been deployed yet
func (v *ContractValidatorWallet) AddressOrZero() common.Address {
	addr := v.address.Load()
	if addr == nil {
		return common.Address{}
	}
	return *addr
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

// nil value == 0 value
func (v *ContractValidatorWallet) getAuth(ctx context.Context, value *big.Int) (*bind.TransactOpts, error) {
	newAuth := *v.auth
	newAuth.Context = ctx
	newAuth.Value = value
	nonce, err := v.L1Client().NonceAt(ctx, v.auth.From, nil)
	if err != nil {
		return nil, err
	}
	newAuth.Nonce = new(big.Int).SetUint64(nonce)
	return &newAuth, nil
}

func (v *ContractValidatorWallet) executeTransaction(ctx context.Context, tx *types.Transaction, gasRefunder common.Address) (*types.Transaction, error) {
	auth, err := v.getAuth(ctx, tx.Value())
	if err != nil {
		return nil, err
	}
	data, err := validatorABI.Pack("executeTransactionWithGasRefunder", gasRefunder, tx.Data(), *tx.To(), tx.Value())
	if err != nil {
		return nil, fmt.Errorf("packing arguments for executeTransactionWithGasRefunder: %w", err)
	}
	gas, err := v.gasForTxData(ctx, auth, data)
	if err != nil {
		return nil, fmt.Errorf("getting gas for tx data: %w", err)
	}
	return v.dataPoster.PostTransaction(ctx, time.Now(), auth.Nonce.Uint64(), nil, *v.Address(), data, gas, auth.Value)
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
	if v.address.Load() == nil {
		auth, err := v.getAuth(ctx, nil)
		if err != nil {
			return err
		}
		addr, err := GetValidatorWalletContract(ctx, v.walletFactoryAddr, v.rollupFromBlock, auth, v.l1Reader, createIfMissing)
		if err != nil {
			return err
		}
		if addr == nil {
			return nil
		}
		v.address.Store(addr)
		if v.onWalletCreated != nil {
			v.onWalletCreated(*addr)
		}
	}
	con, err := rollupgen.NewValidatorWallet(*v.Address(), v.l1Reader.Client())
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

	balanceInContract, err := v.l1Reader.Client().BalanceAt(ctx, *v.Address(), nil)
	if err != nil {
		return nil, err
	}

	callValue := new(big.Int).Sub(totalAmount, balanceInContract)
	if callValue.Sign() < 0 {
		callValue.SetInt64(0)
	}
	auth, err := v.getAuth(ctx, callValue)
	if err != nil {
		return nil, err
	}
	txData, err := validatorABI.Pack("executeTransactionsWithGasRefunder", gasRefunder, data, dest, amount)
	if err != nil {
		return nil, fmt.Errorf("packing arguments for executeTransactionWithGasRefunder: %w", err)
	}
	gas, err := v.gasForTxData(ctx, auth, txData)
	if err != nil {
		return nil, fmt.Errorf("getting gas for tx data: %w", err)
	}
	arbTx, err := v.dataPoster.PostTransaction(ctx, time.Now(), auth.Nonce.Uint64(), nil, *v.Address(), txData, gas, auth.Value)
	if err != nil {
		return nil, err
	}
	builder.transactions = nil
	return arbTx, nil
}

func (v *ContractValidatorWallet) estimateGas(ctx context.Context, value *big.Int, data []byte) (uint64, error) {
	h, err := v.l1Reader.LastHeader(ctx)
	if err != nil {
		return 0, fmt.Errorf("getting the last header: %w", err)
	}
	gasFeeCap := new(big.Int).Mul(h.BaseFee, big.NewInt(2))
	gasFeeCap = arbmath.BigMax(gasFeeCap, arbmath.FloatToBig(params.GWei))

	gasTipCap, err := v.l1Reader.Client().SuggestGasTipCap(ctx)
	if err != nil {
		return 0, fmt.Errorf("getting suggested gas tip cap: %w", err)
	}
	g, err := v.l1Reader.Client().EstimateGas(
		ctx,
		ethereum.CallMsg{
			From:      v.auth.From,
			To:        v.Address(),
			Value:     value,
			Data:      data,
			GasFeeCap: gasFeeCap,
			GasTipCap: gasTipCap,
		},
	)
	if err != nil {
		return 0, fmt.Errorf("estimating gas: %w", err)
	}
	return g + v.getExtraGas(), nil
}

func (v *ContractValidatorWallet) TimeoutChallenges(ctx context.Context, challenges []uint64) (*types.Transaction, error) {
	auth, err := v.getAuth(ctx, nil)
	if err != nil {
		return nil, err
	}
	data, err := validatorABI.Pack("timeoutChallenges", v.challengeManagerAddress, challenges)
	if err != nil {
		return nil, fmt.Errorf("packing arguments for timeoutChallenges: %w", err)
	}
	gas, err := v.gasForTxData(ctx, auth, data)
	if err != nil {
		return nil, fmt.Errorf("getting gas for tx data: %w", err)
	}
	return v.dataPoster.PostTransaction(ctx, time.Now(), auth.Nonce.Uint64(), nil, *v.Address(), data, gas, auth.Value)
}

// gasForTxData returns auth.GasLimit if it's nonzero, otherwise returns estimate.
func (v *ContractValidatorWallet) gasForTxData(ctx context.Context, auth *bind.TransactOpts, data []byte) (uint64, error) {
	if auth.GasLimit != 0 {
		return auth.GasLimit, nil
	}
	return v.estimateGas(ctx, auth.Value, data)
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

func (w *ContractValidatorWallet) Start(ctx context.Context) {
	w.dataPoster.Start(ctx)
}

func (b *ContractValidatorWallet) StopAndWait() {
	b.dataPoster.StopAndWait()
}

func (b *ContractValidatorWallet) DataPoster() *dataposter.DataPoster {
	return b.dataPoster
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
		return nil, err
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
		return nil, err
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
		return nil, err
	}
	log.Info("created validator smart contract wallet", "address", ev.WalletAddress)
	return &ev.WalletAddress, nil
}
