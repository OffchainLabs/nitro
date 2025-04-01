package timeboost

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/ethclient/simulated"
	"github.com/ethereum/go-ethereum/node"

	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/solgen/go/express_lane_auctiongen"
	"github.com/offchainlabs/nitro/solgen/go/mocksgen"
	"github.com/offchainlabs/nitro/timeboost/bindings"
)

type auctionSetup struct {
	chainId                *big.Int
	expressLaneAuctionAddr common.Address
	expressLaneAuction     *express_lane_auctiongen.ExpressLaneAuction
	erc20Addr              common.Address
	erc20Contract          *bindings.MockERC20
	initialTimestamp       time.Time
	roundDuration          time.Duration
	expressLaneAddr        common.Address
	beneficiaryAddr        common.Address
	accounts               []*testAccount
	backend                *simulated.Backend
	endpoint               string
}

func setupAuctionTest(t testing.TB, ctx context.Context) *auctionSetup {
	accs, backend, endpoint := setupAccounts(t, 10)

	go func() {
		tick := time.NewTicker(time.Second)
		defer tick.Stop()
		for {
			select {
			case <-tick.C:
				backend.Commit()
			case <-ctx.Done():
				return
			}
		}
	}()

	opts := accs[0].txOpts
	chainId, err := backend.Client().ChainID(ctx)
	require.NoError(t, err)

	// Deploy the token as a mock erc20.
	erc20Addr, tx, erc20, err := bindings.DeployMockERC20(opts, backend.Client())
	require.NoError(t, err)
	if _, err = bind.WaitMined(ctx, backend.Client(), tx); err != nil {
		t.Fatal(err)
	}
	tx, err = erc20.Initialize(opts, "LANE", "LNE", 18)
	require.NoError(t, err)
	if _, err = bind.WaitMined(ctx, backend.Client(), tx); err != nil {
		t.Fatal(err)
	}

	// Mint 10 wei tokens to all accounts.
	mintTokens(ctx, opts, backend, accs, erc20)

	// Check account balances.
	bal, err := erc20.BalanceOf(&bind.CallOpts{}, accs[0].accountAddr)
	require.NoError(t, err)
	t.Log("Account seeded with ERC20 token balance =", bal.String())

	// Deploy the express lane auction contract.
	auctionContractAddr, tx, _, err := express_lane_auctiongen.DeployExpressLaneAuction(
		opts, backend.Client(),
	)
	require.NoError(t, err)
	if _, err = bind.WaitMined(ctx, backend.Client(), tx); err != nil {
		t.Fatal(err)
	}
	proxyAddr, tx, _, err := mocksgen.DeploySimpleProxy(opts, backend.Client(), auctionContractAddr)
	require.NoError(t, err)
	if _, err = bind.WaitMined(ctx, backend.Client(), tx); err != nil {
		t.Fatal(err)
	}
	auctionContract, err := express_lane_auctiongen.NewExpressLaneAuction(proxyAddr, backend.Client())
	require.NoError(t, err)

	expressLaneAddr := common.HexToAddress("0x2424242424242424242424242424242424242424")

	// Calculate the number of seconds until the next minute
	// and the next timestamp that is a multiple of a minute.
	now := time.Now()
	roundDuration := time.Minute
	waitTime := roundDuration - time.Duration(now.Second())*time.Second - time.Duration(now.Nanosecond())
	initialTime := now.Add(waitTime)
	initialTimestamp := big.NewInt(initialTime.Unix())
	t.Logf("Initial timestamp for express lane auctions: %v", initialTime)

	// Deploy the auction manager contract.
	auctioneer := opts.From
	beneficiary := opts.From
	biddingToken := erc20Addr
	bidRoundSeconds := uint64(60)
	auctionClosingSeconds := uint64(15)
	reserveSubmissionSeconds := uint64(15)
	minReservePrice := big.NewInt(1) // 1 wei.
	roleAdmin := opts.From
	tx, err = auctionContract.Initialize(
		opts,
		express_lane_auctiongen.InitArgs{
			Auctioneer:   auctioneer,
			BiddingToken: biddingToken,
			Beneficiary:  beneficiary,
			RoundTimingInfo: express_lane_auctiongen.RoundTimingInfo{
				OffsetTimestamp:          initialTimestamp.Int64(),
				RoundDurationSeconds:     bidRoundSeconds,
				AuctionClosingSeconds:    auctionClosingSeconds,
				ReserveSubmissionSeconds: reserveSubmissionSeconds,
			},
			MinReservePrice:       minReservePrice,
			AuctioneerAdmin:       roleAdmin,
			MinReservePriceSetter: roleAdmin,
			ReservePriceSetter:    roleAdmin,
			BeneficiarySetter:     roleAdmin,
			RoundTimingSetter:     roleAdmin,
			MasterAdmin:           roleAdmin,
		},
	)
	require.NoError(t, err)
	if _, err = bind.WaitMined(ctx, backend.Client(), tx); err != nil {
		t.Fatal(err)
	}
	return &auctionSetup{
		chainId:                chainId,
		expressLaneAuctionAddr: proxyAddr,
		expressLaneAuction:     auctionContract,
		erc20Addr:              erc20Addr,
		erc20Contract:          erc20,
		initialTimestamp:       now,
		roundDuration:          time.Minute,
		expressLaneAddr:        expressLaneAddr,
		beneficiaryAddr:        beneficiary,
		accounts:               accs,
		backend:                backend,
		endpoint:               endpoint,
	}
}

func setupBidderClient(
	t testing.TB, ctx context.Context, account *testAccount, testSetup *auctionSetup, bidValidatorEndpoint string,
) *BidderClient {
	cfgFetcher := func() *BidderClientConfig {
		return &BidderClientConfig{
			AuctionContractAddress: testSetup.expressLaneAuctionAddr.Hex(),
			BidValidatorEndpoint:   bidValidatorEndpoint,
			ArbitrumNodeEndpoint:   testSetup.endpoint,
			Wallet: genericconf.WalletConfig{
				PrivateKey: fmt.Sprintf("%x", account.privKey.D.Bytes()),
			},
		}
	}
	bc, err := NewBidderClient(
		ctx,
		cfgFetcher,
	)
	require.NoError(t, err)
	bc.Start(ctx)

	// Approve spending by the express lane auction contract and beneficiary.
	maxUint256 := big.NewInt(1)
	maxUint256.Lsh(maxUint256, 256).Sub(maxUint256, big.NewInt(1))
	tx, err := testSetup.erc20Contract.Approve(
		account.txOpts, testSetup.expressLaneAuctionAddr, maxUint256,
	)
	require.NoError(t, err)
	if _, err = bind.WaitMined(ctx, testSetup.backend.Client(), tx); err != nil {
		t.Fatal(err)
	}
	tx, err = testSetup.erc20Contract.Approve(
		account.txOpts, testSetup.beneficiaryAddr, maxUint256,
	)
	require.NoError(t, err)
	if _, err = bind.WaitMined(ctx, testSetup.backend.Client(), tx); err != nil {
		t.Fatal(err)
	}
	return bc
}

type testAccount struct {
	accountAddr common.Address
	privKey     *ecdsa.PrivateKey
	txOpts      *bind.TransactOpts
}

func setupAccounts(t testing.TB, numAccounts uint64) ([]*testAccount, *simulated.Backend, string) {
	genesis := make(core.GenesisAlloc)
	gasLimit := uint64(100000000)

	accs := make([]*testAccount, numAccounts)
	for i := uint64(0); i < numAccounts; i++ {
		privKey, err := crypto.GenerateKey()
		if err != nil {
			panic(err)
		}
		addr := crypto.PubkeyToAddress(privKey.PublicKey)
		chainID := big.NewInt(1337)
		txOpts, err := bind.NewKeyedTransactorWithChainID(privKey, chainID)
		if err != nil {
			panic(err)
		}
		startingBalance, _ := new(big.Int).SetString(
			"100000000000000000000000000000000000000",
			10,
		)
		genesis[addr] = core.GenesisAccount{Balance: startingBalance}
		accs[i] = &testAccount{
			accountAddr: addr,
			txOpts:      txOpts,
			privKey:     privKey,
		}
	}
	randPort := getRandomPort(t)
	withRPC := func(n *node.Config, _ *ethconfig.Config) {
		n.HTTPHost = "localhost"
		n.HTTPPort = randPort
		n.HTTPModules = []string{"eth", "net", "web3", "debug", "personal"}
	}
	backend := simulated.NewBackend(genesis, simulated.WithBlockGasLimit(gasLimit), withRPC)
	return accs, backend, fmt.Sprintf("http://localhost:%d", randPort)
}

func mintTokens(ctx context.Context,
	opts *bind.TransactOpts,
	backend *simulated.Backend,
	accs []*testAccount,
	erc20 *bindings.MockERC20,
) {
	for i := 0; i < len(accs); i++ {
		tx, err := erc20.Mint(opts, accs[i].accountAddr, big.NewInt(100))
		if err != nil {
			panic(err)
		}
		if _, err = bind.WaitMined(ctx, backend.Client(), tx); err != nil {
			panic(err)
		}
	}
}
