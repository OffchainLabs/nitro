package timeboost

import (
	"context"
	"fmt"
	"math/big"

	"github.com/pkg/errors"
	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/cmd/util"
	"github.com/offchainlabs/nitro/solgen/go/express_lane_auctiongen"
	"github.com/offchainlabs/nitro/timeboost/bindings"
	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/signature"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

type BidderClientConfigFetcher func() *BidderClientConfig

type BidderClientConfig struct {
	Wallet                 genericconf.WalletConfig `koanf:"wallet"`
	ArbitrumNodeEndpoint   string                   `koanf:"arbitrum-node-endpoint"`
	BidValidatorEndpoint   string                   `koanf:"bid-validator-endpoint"`
	AuctionContractAddress string                   `koanf:"auction-contract-address"`
	DepositGwei            int                      `koanf:"deposit-gwei"`
	BidGwei                int                      `koanf:"bid-gwei"`
}

var DefaultBidderClientConfig = BidderClientConfig{
	ArbitrumNodeEndpoint: "http://localhost:8547",
	BidValidatorEndpoint: "http://localhost:9372",
}

var TestBidderClientConfig = BidderClientConfig{
	ArbitrumNodeEndpoint: "http://localhost:8547",
	BidValidatorEndpoint: "http://localhost:9372",
}

func BidderClientConfigAddOptions(f *pflag.FlagSet) {
	genericconf.WalletConfigAddOptions("wallet", f, "wallet for bidder")
	f.String("arbitrum-node-endpoint", DefaultBidderClientConfig.ArbitrumNodeEndpoint, "arbitrum node RPC http endpoint")
	f.String("bid-validator-endpoint", DefaultBidderClientConfig.BidValidatorEndpoint, "bid validator http endpoint")
	f.String("auction-contract-address", DefaultBidderClientConfig.AuctionContractAddress, "express lane auction contract address")
	f.Int("deposit-gwei", DefaultBidderClientConfig.DepositGwei, "deposit amount in gwei to take from bidder's account and send to auction contract")
	f.Int("bid-gwei", DefaultBidderClientConfig.BidGwei, "bid amount in gwei, bidder must have already deposited enough into the auction contract")
}

type BidderClient struct {
	stopwaiter.StopWaiter
	chainId                *big.Int
	auctionContractAddress common.Address
	biddingTokenAddress    common.Address
	txOpts                 *bind.TransactOpts
	client                 *ethclient.Client
	signer                 signature.DataSignerFunc
	auctionContract        *express_lane_auctiongen.ExpressLaneAuction
	biddingTokenContract   *bindings.MockERC20
	auctioneerClient       *rpc.Client
	roundTimingInfo        RoundTimingInfo
	domainValue            []byte
}

func NewBidderClient(
	ctx context.Context,
	configFetcher BidderClientConfigFetcher,
) (*BidderClient, error) {
	cfg := configFetcher()
	_ = cfg.BidGwei     // These fields are used from cmd/bidder-client
	_ = cfg.DepositGwei // this marks them as used for the linter.
	if cfg.AuctionContractAddress == "" {
		return nil, fmt.Errorf("auction contract address cannot be empty")
	}
	auctionContractAddr := common.HexToAddress(cfg.AuctionContractAddress)
	client, err := rpc.DialContext(ctx, cfg.ArbitrumNodeEndpoint)
	if err != nil {
		return nil, err
	}
	arbClient := ethclient.NewClient(client)
	chainId, err := arbClient.ChainID(ctx)
	if err != nil {
		return nil, err
	}
	auctionContract, err := express_lane_auctiongen.NewExpressLaneAuction(auctionContractAddr, arbClient)
	if err != nil {
		return nil, err
	}
	rawRoundTimingInfo, err := auctionContract.RoundTimingInfo(&bind.CallOpts{
		Context: ctx,
	})
	if err != nil {
		return nil, err
	}
	roundTimingInfo, err := NewRoundTimingInfo(rawRoundTimingInfo)
	if err != nil {
		return nil, err
	}
	txOpts, signer, err := util.OpenWallet("bidder-client", &cfg.Wallet, chainId)
	if err != nil {
		return nil, errors.Wrap(err, "opening wallet")
	}

	biddingTokenAddr, err := auctionContract.BiddingToken(&bind.CallOpts{
		Context: ctx,
	})
	if err != nil {
		return nil, errors.Wrap(err, "fetching bidding token")
	}
	biddingTokenContract, err := bindings.NewMockERC20(biddingTokenAddr, arbClient)
	if err != nil {
		return nil, errors.Wrap(err, "creating bindings to bidding token contract")
	}

	bidValidatorClient, err := rpc.DialContext(ctx, cfg.BidValidatorEndpoint)
	if err != nil {
		return nil, err
	}
	return &BidderClient{
		chainId:                chainId,
		auctionContractAddress: auctionContractAddr,
		biddingTokenAddress:    biddingTokenAddr,
		client:                 arbClient,
		txOpts:                 txOpts,
		signer:                 signer,
		auctionContract:        auctionContract,
		biddingTokenContract:   biddingTokenContract,
		auctioneerClient:       bidValidatorClient,
		roundTimingInfo:        *roundTimingInfo,
		domainValue:            domainValue,
	}, nil
}

func (bd *BidderClient) Start(ctx_in context.Context) {
	bd.StopWaiter.Start(ctx_in, bd)
}

// Deposit into the auction contract for the account configured by the BidderClient wallet.
// Handles approving the auction contract to spend the erc20 on behalf of the account.
func (bd *BidderClient) Deposit(ctx context.Context, amount *big.Int) error {
	allowance, err := bd.biddingTokenContract.Allowance(&bind.CallOpts{
		Context: ctx,
	}, bd.txOpts.From, bd.auctionContractAddress)
	if err != nil {
		return err
	}

	if amount.Cmp(allowance) > 0 {
		log.Info("Spend allowance of bidding token from auction contract is insufficient, increasing allowance", "from", bd.txOpts.From, "auctionContract", bd.auctionContractAddress, "biddingToken", bd.biddingTokenAddress, "amount", amount.Int64())
		//		defecit := arbmath.BigSub(allowance, amount)
		tx, err := bd.biddingTokenContract.Approve(bd.txOpts, bd.auctionContractAddress, amount)
		if err != nil {
			return err
		}
		receipt, err := bind.WaitMined(ctx, bd.client, tx)
		if err != nil {
			return err
		}
		if receipt.Status != types.ReceiptStatusSuccessful {
			return errors.New("approval failed")
		}
	}

	tx, err := bd.auctionContract.Deposit(bd.txOpts, amount)
	if err != nil {
		return err
	}
	receipt, err := bind.WaitMined(ctx, bd.client, tx)
	if err != nil {
		return err
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return errors.New("deposit failed")
	}
	return nil
}

func (bd *BidderClient) Bid(
	ctx context.Context, amount *big.Int, expressLaneController common.Address,
) (*Bid, error) {
	if (expressLaneController == common.Address{}) {
		expressLaneController = bd.txOpts.From
	}

	domainSeparator, err := bd.auctionContract.DomainSeparator(&bind.CallOpts{
		Context: ctx,
	})
	if err != nil {
		return nil, err
	}
	newBid := &Bid{
		ChainId:                bd.chainId,
		ExpressLaneController:  expressLaneController,
		AuctionContractAddress: bd.auctionContractAddress,
		Round:                  bd.roundTimingInfo.RoundNumber() + 1,
		Amount:                 amount,
	}
	bidHash, err := newBid.ToEIP712Hash(domainSeparator)
	if err != nil {
		return nil, err
	}

	sig, err := bd.signer(bidHash.Bytes())
	if err != nil {
		return nil, err
	}
	sig[64] += 27

	newBid.Signature = sig

	promise := bd.submitBid(newBid)
	if _, err := promise.Await(ctx); err != nil {
		return nil, err
	}
	return newBid, nil
}

func (bd *BidderClient) submitBid(bid *Bid) containers.PromiseInterface[struct{}] {
	return stopwaiter.LaunchPromiseThread[struct{}](bd, func(ctx context.Context) (struct{}, error) {
		err := bd.auctioneerClient.CallContext(ctx, nil, "auctioneer_submitBid", bid.ToJson())
		return struct{}{}, err
	})
}
