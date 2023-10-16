package arbnode

import (
	"context"
	"errors"

	"github.com/OffchainLabs/bold/solgen/go/rollupgen"
	"github.com/OffchainLabs/bold/testing/setup"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/util/headerreader"
)

func DeployBOLDOnL1(ctx context.Context, parentChainReader *headerreader.HeaderReader, deployAuth *bind.TransactOpts, batchPoster common.Address, authorizeValidators uint64, config rollupgen.Config) (*setup.RollupAddresses, error) {
	if config.WasmModuleRoot == (common.Hash{}) {
		return nil, errors.New("no machine specified")
	}
	addresses, err := setup.DeployFullRollupStack(
		ctx,
		parentChainReader.Client(),
		deployAuth,
		deployAuth.From,
		config,
		false, // do not use mock bridge.
		false, // do not use a mock one step prover
	)
	if err != nil {
		return nil, err
	}
	return addresses, nil
}
