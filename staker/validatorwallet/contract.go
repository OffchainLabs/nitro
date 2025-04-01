// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package validatorwallet

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbnode/dataposter"
	"github.com/offchainlabs/nitro/solgen/go/rollupgen"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/headerreader"
)

var (
	validatorABI              abi.ABI
	validatorWalletCreatorABI abi.ABI
	walletCreatedID           common.Hash
)

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
	validatorWalletCreatorABI = parsedValidatorWalletCreator
	walletCreatedID = parsedValidatorWalletCreator.Events["WalletCreated"].ID
}

type Contract struct {
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
	populateWalletMutex     sync.Mutex
}

func NewContract(dp *dataposter.DataPoster, address *common.Address, walletFactoryAddr, rollupAddress common.Address, l1Reader *headerreader.HeaderReader, auth *bind.TransactOpts, rollupFromBlock int64, onWalletCreated func(common.Address),
	getExtraGas func() uint64) (*Contract, error) {
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
	wallet := &Contract{
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

func (v *Contract) validateWallet(ctx context.Context) error {
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

func (v *Contract) Initialize(ctx context.Context) error {
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
func (v *Contract) Address() *common.Address {
	return v.address.Load()
}

// May be zero if the wallet hasn't been deployed yet
func (v *Contract) AddressOrZero() common.Address {
	addr := v.address.Load()
	if addr == nil {
		return common.Address{}
	}
	return *addr
}

func (v *Contract) TxSenderAddress() *common.Address {
	if v.auth == nil {
		return nil
	}
	return &v.auth.From
}

func (v *Contract) From() common.Address {
	if v.auth == nil {
		return common.Address{}
	}
	return v.auth.From
}

func (v *Contract) executeTransaction(ctx context.Context, tx *types.Transaction, gasRefunder common.Address) (*types.Transaction, error) {
	data, err := validatorABI.Pack("executeTransactionWithGasRefunder", gasRefunder, tx.Data(), *tx.To(), tx.Value())
	if err != nil {
		return nil, fmt.Errorf("packing arguments for executeTransactionWithGasRefunder: %w", err)
	}
	gas, err := v.gasForTxData(ctx, data, tx.Value())
	if err != nil {
		return nil, fmt.Errorf("getting gas for tx data: %w", err)
	}
	return v.dataPoster.PostSimpleTransaction(ctx, *v.Address(), data, gas, tx.Value())
}

func createWalletContract(
	ctx context.Context,
	l1Reader *headerreader.HeaderReader,
	from common.Address,
	dataPoster *dataposter.DataPoster,
	getExtraGas func() uint64,
	validatorWalletFactoryAddr common.Address,
) (*types.Transaction, error) {
	var initialExecutorAllowedDests []common.Address
	txData, err := validatorWalletCreatorABI.Pack("createWallet", initialExecutorAllowedDests)
	if err != nil {
		return nil, err
	}

	gas, err := gasForTxData(
		ctx,
		l1Reader,
		from,
		&validatorWalletFactoryAddr,
		txData,
		common.Big0,
		getExtraGas,
	)
	if err != nil {
		return nil, fmt.Errorf("getting gas for tx data when creating validator wallet, validatorWalletFactory=%v: %w", validatorWalletFactoryAddr, err)
	}

	return dataPoster.PostSimpleTransaction(ctx, validatorWalletFactoryAddr, txData, gas, common.Big0)
}

func (v *Contract) populateWallet(ctx context.Context, createIfMissing bool) error {
	v.populateWalletMutex.Lock()
	defer v.populateWalletMutex.Unlock()
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
		// By passing v.dataPoster as a parameter to GetValidatorWalletContract we force to create a validator wallet through the Staker's DataPoster object.
		// DataPoster keeps in its internal state information related to the transactions sent through it, which is used to infer the expected nonce in a transaction for example.
		// If a transaction is sent using the Staker's DataPoster key, but not through the Staker's DataPoster object, DataPoster's internal state will be outdated, which can compromise the expected nonce inference.
		addr, err := GetValidatorWalletContract(ctx, v.walletFactoryAddr, v.rollupFromBlock, v.l1Reader, createIfMissing, v.dataPoster, v.getExtraGas)
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

func (v *Contract) ExecuteTransactions(ctx context.Context, txes []*types.Transaction, gasRefunder common.Address) (*types.Transaction, error) {
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
	txData, err := validatorABI.Pack("executeTransactionsWithGasRefunder", gasRefunder, data, dest, amount)
	if err != nil {
		return nil, fmt.Errorf("packing arguments for executeTransactionWithGasRefunder: %w", err)
	}
	gas, err := v.gasForTxData(ctx, txData, callValue)
	if err != nil {
		return nil, fmt.Errorf("getting gas for tx data: %w", err)
	}
	arbTx, err := v.dataPoster.PostSimpleTransaction(ctx, *v.Address(), txData, gas, callValue)
	if err != nil {
		return nil, err
	}
	return arbTx, nil
}

func gasForTxData(ctx context.Context, l1Reader *headerreader.HeaderReader, from common.Address, to *common.Address, data []byte, value *big.Int, getExtraGas func() uint64) (uint64, error) {
	h, err := l1Reader.LastHeader(ctx)
	if err != nil {
		return 0, fmt.Errorf("getting the last header: %w", err)
	}
	gasFeeCap := new(big.Int).Mul(h.BaseFee, big.NewInt(2))
	gasFeeCap = arbmath.BigMax(gasFeeCap, arbmath.FloatToBig(params.GWei))

	gasTipCap, err := l1Reader.Client().SuggestGasTipCap(ctx)
	if err != nil {
		return 0, fmt.Errorf("getting suggested gas tip cap: %w", err)
	}
	gasFeeCap.Add(gasFeeCap, gasTipCap)
	g, err := l1Reader.Client().EstimateGas(
		ctx,
		ethereum.CallMsg{
			From:      from,
			To:        to,
			Value:     value,
			Data:      data,
			GasFeeCap: gasFeeCap,
			GasTipCap: gasTipCap,
		},
	)
	if err != nil {
		return 0, fmt.Errorf("estimating gas: %w", err)
	}
	return g + getExtraGas(), nil
}

func (v *Contract) gasForTxData(ctx context.Context, data []byte, value *big.Int) (uint64, error) {
	return gasForTxData(ctx, v.l1Reader, v.From(), v.Address(), data, value, v.getExtraGas)
}

func (v *Contract) TimeoutChallenges(ctx context.Context, challenges []uint64) (*types.Transaction, error) {
	data, err := validatorABI.Pack("timeoutChallenges", v.challengeManagerAddress, challenges)
	if err != nil {
		return nil, fmt.Errorf("packing arguments for timeoutChallenges: %w", err)
	}
	gas, err := v.gasForTxData(ctx, data, common.Big0)
	if err != nil {
		return nil, fmt.Errorf("getting gas for tx data: %w", err)
	}
	return v.dataPoster.PostSimpleTransaction(ctx, *v.Address(), data, gas, common.Big0)
}

func (v *Contract) L1Client() *ethclient.Client {
	return v.l1Reader.Client()
}

func (v *Contract) RollupAddress() common.Address {
	return v.rollupAddress
}

func (v *Contract) ChallengeManagerAddress() common.Address {
	return v.challengeManagerAddress
}

func (v *Contract) TestTransactions(ctx context.Context, txs []*types.Transaction) error {
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

func (v *Contract) CanBatchTxs() bool {
	return true
}

func (v *Contract) AuthIfEoa() *bind.TransactOpts {
	return nil
}

func (w *Contract) Start(ctx context.Context) {
	w.dataPoster.Start(ctx)
}

func (b *Contract) StopAndWait() {
	b.dataPoster.StopAndWait()
}

func (b *Contract) DataPoster() *dataposter.DataPoster {
	return b.dataPoster
}

// Exported for testing
func (b *Contract) GetExtraGas() func() uint64 {
	return b.getExtraGas
}

func GetValidatorWalletContract(
	ctx context.Context,
	validatorWalletFactoryAddr common.Address,
	fromBlock int64,
	l1Reader *headerreader.HeaderReader,
	createIfMissing bool,
	dataPoster *dataposter.DataPoster,
	getExtraGas func() uint64,
) (*common.Address, error) {
	client := l1Reader.Client()
	transactAuth := dataPoster.Auth()

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
		Topics:    [][]common.Hash{{walletCreatedID}, nil, {common.BytesToHash(transactAuth.From.Bytes())}},
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

	tx, err := createWalletContract(ctx, l1Reader, transactAuth.From, dataPoster, getExtraGas, validatorWalletFactoryAddr)
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
