package arbtest

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/solgen/go/localgen"
)

func TestSelfDestruct(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	cleanup := builder.Build(t)
	defer cleanup()

	builder.L2Info.GenerateAccount("destination")
	destination := builder.L2Info.GetAddress("destination")
	builder.L2Info.GenerateAccount("self_destruct")
	auth := builder.L2Info.GetDefaultTransactOpts("self_destruct", ctx)
	auth.GasLimit = 32000000
	balance := big.NewInt(params.Ether)
	balance.Mul(balance, big.NewInt(100))
	builder.L2.TransferBalance(t, "Faucet", "self_destruct", balance, builder.L2Info)

	// Test self-destruct with recipient same as the contract (contract is created and destroyed in the same transaction)
	auth.Value = big.NewInt(params.Ether)
	_, tx, _, err := localgen.DeploySelfDestructInConstructorWithoutDestination(&auth, builder.L2.Client)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	// Test self-destruct with recipient different from the contract (contract is created and destroyed in the same transaction)
	auth.Value = big.NewInt(params.Ether)
	_, tx, _, err = localgen.DeploySelfDestructInConstructorWithDestination(&auth, builder.L2.Client, destination)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	// Test self-destruct with recipient same as the contract (contract is created and destroyed in different transaction)
	auth.Value = big.NewInt(params.Ether)
	_, tx, selfDestructOutsideConstructor, err := localgen.DeploySelfDestructOutsideConstructor(&auth, builder.L2.Client)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
	auth.Value = nil
	tx, err = selfDestructOutsideConstructor.SelfDestructWithoutDestination(&auth)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	// Test self-destruct with recipient same as the contract (contract is created and destroyed in the different transaction)
	auth.Value = big.NewInt(params.Ether)
	_, tx, selfDestructOutsideConstructor, err = localgen.DeploySelfDestructOutsideConstructor(&auth, builder.L2.Client)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
	auth.Value = nil
	tx, err = selfDestructOutsideConstructor.SelfDestructWithDestination(&auth, destination)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
}
