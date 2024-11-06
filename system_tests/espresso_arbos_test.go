package arbtest

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
)

func EspressoArbOSTestChainConfig() *params.ChainConfig {
	return &params.ChainConfig{
		ChainID:             big.NewInt(412346),
		HomesteadBlock:      big.NewInt(0),
		DAOForkBlock:        nil,
		DAOForkSupport:      true,
		EIP150Block:         big.NewInt(0),
		EIP155Block:         big.NewInt(0),
		EIP158Block:         big.NewInt(0),
		ByzantiumBlock:      big.NewInt(0),
		ConstantinopleBlock: big.NewInt(0),
		PetersburgBlock:     big.NewInt(0),
		IstanbulBlock:       big.NewInt(0),
		MuirGlacierBlock:    big.NewInt(0),
		BerlinBlock:         big.NewInt(0),
		LondonBlock:         big.NewInt(0),
		ArbitrumChainParams: EspressoTestChainParams(),
		Clique: &params.CliqueConfig{
			Period: 0,
			Epoch:  0,
		},
	}
}
func EspressoTestChainParams() params.ArbitrumChainParams {
	return params.ArbitrumChainParams{
		EnableArbOS:               true,
		AllowDebugPrecompiles:     true,
		DataAvailabilityCommittee: false,
		InitialArbOSVersion:       31,
		InitialChainOwner:         common.Address{},
		EnableEspresso: 		   false,
	}
}

func waitForConfigUpdate(t *testing.T, ctx context.Context, builder *NodeBuilder) error{

    return waitForWith(t, ctx, 120*time.Second, 1*time.Second, func() bool{
      newArbOSConfig, err := builder.L2.ExecNode.GetArbOSConfigAtHeight(0)
      Require(t, err)

      if newArbOSConfig.ArbitrumChainParams.EnableEspresso != false{
        return false
      }
      Require(t,err)
      return true
    })
}

func TestEspressoArbOSConfig(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder, cleanup := createL1AndL2Node(ctx, t)
	defer cleanup()

	err := waitForL1Node(t, ctx)
	Require(t, err)

	cleanEspresso := runEspresso(t, ctx)
	defer cleanEspresso()

	// wait for the builder
	err = waitForEspressoNode(t, ctx)
	Require(t, err)

	l2Node := builder.L2

	// Wait for the initial message
	expected := arbutil.MessageIndex(1)
	err = waitFor(t, ctx, func() bool {
		msgCnt, err := l2Node.ConsensusNode.TxStreamer.GetMessageCount()
		if err != nil {
			panic(err)
		}

		validatedCnt := l2Node.ConsensusNode.BlockValidator.Validated(t)
		return msgCnt >= expected && validatedCnt >= expected
	})
	Require(t, err)
  

  initialArbOSConfig, err := builder.L2.ExecNode.GetArbOSConfigAtHeight(0)
  Require(t,err)

  //assert that espresso is initially enabled
  if initialArbOSConfig.ArbitrumChainParams.EnableEspresso != true{
    err = fmt.Errorf("Initial config should have EnableEspresso == true!")
    
  } 
  Require(t,err)

  newArbOwner, err := precompilesgen.NewArbOwner(common.HexToAddress("0x070"), builder.L2.Client)
  Require(t, err)

  newArbDebug, err := precompilesgen.NewArbDebug(common.HexToAddress("0xff"), builder.L2.Client)
  Require(t, err)
  
  l2auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)

  _, err = newArbDebug.BecomeChainOwner(&l2auth)
  Require(t, err)
  chainConfig, err := json.Marshal(EspressoArbOSTestChainConfig())
  Require(t, err)
  
  chainConfigString := string(chainConfig)

  _, err = newArbOwner.SetChainConfig(&l2auth, chainConfigString)
  Require(t, err)
  // check if chain config is updated TODO replace this with a wait for with to poll for some time potentially
  
  waitForConfigUpdate(t, ctx, builder)
}
