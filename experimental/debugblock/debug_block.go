//go:build debugblock

package debugblock

import (
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/util"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/spf13/pflag"
)

func ConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".overwrite-chain-config", ConfigDefault.OverwriteChainConfig, "DANGEROUS! overwrites chain when opening existing database; chain debug mode will be enabled")
	f.String(prefix+".debug-address", ConfigDefault.DebugAddress, "DANGEROUS! address of debug account to be pre-funded")
	f.Uint64(prefix+".debug-blocknum", ConfigDefault.DebugBlockNum, "DANGEROUS! block number of injected debug block")
}

func (c *Config) Validate() error {
	if c.OverwriteChainConfig {
		log.Warn("DANGER! overwrite-chain-config set, chain config will be over-written")
	}
	if c.DebugAddress != "" && !common.IsHexAddress(c.DebugAddress) {
		return errors.New("invalid debug-address, hex address expected")
	}
	if c.DebugBlockNum != 0 {
		log.Warn("DANGER! debug-blocknum set", "blocknum", c.DebugBlockNum)
	}
	return nil
}

func (c *Config) Apply(chainConfig *params.ChainConfig) {
	if c.OverwriteChainConfig {
		chainConfig.ArbitrumChainParams.AllowDebugPrecompiles = true
		chainConfig.ArbitrumChainParams.DebugAddress = common.HexToAddress(c.DebugAddress)
		chainConfig.ArbitrumChainParams.DebugBlock = c.DebugBlockNum
	}
}

// private key and address of account to be used by PrepareDebugTransaction
func triggerPrivateKeyAndAddress() (*ecdsa.PrivateKey, common.Address, error) {
	key, err := crypto.HexToECDSA("acb2d96fc54f5db4530d6c5a6adfd10964b1b62222d875e08b68b72cc9b9935c")
	if err != nil {
		return nil, common.Address{}, err
	}
	return key, crypto.PubkeyToAddress(key.PublicKey), nil
}

// prepares transaction used to trigger debug block creation
// the transaction needs pre-funding within DebugBlockStateUpdate (executed in the begging of debug block, before the trigger transaction)
func PrepareDebugTransaction(chainConfig *params.ChainConfig, lastHeader *types.Header) *types.Transaction {
	if !chainConfig.DebugMode() {
		return nil
	}
	privateKey, address, err := triggerPrivateKeyAndAddress()
	if err != nil {
		log.Error("debug block: failed to get hardcoded private key and address", "err", err)
		return nil
	}
	transferGas := util.NormalizeL2GasForL1GasInitial(800_000, params.GWei) // include room for L1 costs
	txData := &types.DynamicFeeTx{
		To:        &address,
		Gas:       transferGas,
		GasTipCap: big.NewInt(0),
		GasFeeCap: big.NewInt(params.GWei),
		Value:     big.NewInt(0),
		Nonce:     0,
		Data:      nil,
	}
	nextHeaderNumber := arbmath.BigAdd(lastHeader.Number, common.Big1)
	arbosVersion := types.DeserializeHeaderExtraInformation(lastHeader).ArbOSFormatVersion
	signer := types.MakeSigner(chainConfig, nextHeaderNumber, lastHeader.Time, arbosVersion)
	tx := types.NewTx(txData)
	tx, err = types.SignTx(tx, signer, privateKey)
	if err != nil {
		log.Error("debug block: failed to sign trigger tx", "address", address, "err", err)
		return nil
	}
	log.Warn("debug block: PrepareDebugTransaction", "address", address)
	return tx
}

func DebugBlockStateUpdate(statedb *state.StateDB, expectedBalanceDelta *big.Int, chainConfig *params.ChainConfig) {
	// fund trigger account - used to send a transaction that will trigger this block
	transferGas := util.NormalizeL2GasForL1GasInitial(800_000, params.GWei) // include room for L1 costs
	triggerCost := uint256.MustFromBig(new(big.Int).Mul(big.NewInt(int64(transferGas)), big.NewInt(params.GWei)))
	_, triggerAddress, err := triggerPrivateKeyAndAddress()
	if err != nil {
		log.Error("debug block: failed to get hardcoded address", "err", err)
		return
	}
	statedb.SetBalance(triggerAddress, triggerCost, tracing.BalanceChangeUnspecified)
	expectedBalanceDelta.Add(expectedBalanceDelta, triggerCost.ToBig())

	// fund debug account
	balance := uint256.MustFromBig(new(big.Int).Lsh(big.NewInt(1), 254))
	statedb.SetBalance(chainConfig.ArbitrumChainParams.DebugAddress, balance, tracing.BalanceChangeUnspecified)
	expectedBalanceDelta.Add(expectedBalanceDelta, balance.ToBig())

	// save current chain config to arbos state in case it was changed to enable debug mode and debug block
	// replay binary reads chain config from arbos state, that will enable successful validation of future blocks
	// (debug block will still fail validation if chain config was changed off-chain)
	if serializedChainConfig, err := json.Marshal(chainConfig); err != nil {
		log.Error("debug block: failed to marshal chain config", "err", err)
	} else if arbStateWrite, err := arbosState.OpenSystemArbosState(statedb, nil, false); err != nil {
		log.Error("debug block: failed to open arbos state for writing", "err", err)
	} else if err = arbStateWrite.SetChainConfig(serializedChainConfig); err != nil {
		log.Error("debug block: failed to set chain config in arbos state", "err", err)
	}
	log.Warn("DANGER! Producing debug block and funding debug account", "debugAddress", chainConfig.ArbitrumChainParams.DebugAddress)
}
