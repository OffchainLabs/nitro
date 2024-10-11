package timeboost

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/cmd/util"
	"github.com/offchainlabs/nitro/solgen/go/express_lane_auctiongen"
	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/signature"
	"github.com/offchainlabs/nitro/util/stopwaiter"
	"github.com/pkg/errors"
	"github.com/spf13/pflag"
)

type BidderClientConfigFetcher func() *BidderClientConfig

type BidderClientConfig struct {
	Wallet                 genericconf.WalletConfig `koanf:"wallet"`
	ArbitrumNodeEndpoint   string                   `koanf:"arbitrum-node-endpoint"`
	BidValidatorEndpoint   string                   `koanf:"bid-validator-endpoint"`
	AuctionContractAddress string                   `koanf:"auction-contract-address"`
}

var DefaultBidderClientConfig = BidderClientConfig{
	ArbitrumNodeEndpoint: "http://localhost:8547",
	BidValidatorEndpoint: "http://localhost:9372",
}

var TestBidderClientConfig = BidderClientConfig{
	ArbitrumNodeEndpoint: "http://localhost:8547",
	BidValidatorEndpoint: "http://localhost:9372",
}

func BidderClientConfigAddOptions(prefix string, f *pflag.FlagSet) {
	genericconf.WalletConfigAddOptions(prefix+".wallet", f, "wallet for auctioneer server")
	f.String(prefix+".arbitrum-node-endpoint", DefaultBidderClientConfig.ArbitrumNodeEndpoint, "arbitrum node RPC http endpoint")
	f.String(prefix+".bid-validator-endpoint", DefaultBidderClientConfig.BidValidatorEndpoint, "bid validator http endpoint")
	f.String(prefix+".auction-contract-address", DefaultBidderClientConfig.AuctionContractAddress, "express lane auction contract address")
}

type BidderClient struct {
	stopwaiter.StopWaiter
	chainId                *big.Int
	auctionContractAddress common.Address
	txOpts                 *bind.TransactOpts
	client                 *ethclient.Client
	signer                 signature.DataSignerFunc
	auctionContract        *express_lane_auctiongen.ExpressLaneAuction
	auctioneerClient       *rpc.Client
	initialRoundTimestamp  time.Time
	roundDuration          time.Duration
	domainValue            []byte
}

func NewBidderClient(
	ctx context.Context,
	configFetcher BidderClientConfigFetcher,
) (*BidderClient, error) {
	cfg := configFetcher()
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
	roundTimingInfo, err := auctionContract.RoundTimingInfo(&bind.CallOpts{
		Context: ctx,
	})
	if err != nil {
		return nil, err
	}
	initialTimestamp := time.Unix(int64(roundTimingInfo.OffsetTimestamp), 0)
	roundDuration := time.Duration(roundTimingInfo.RoundDurationSeconds) * time.Second
	txOpts, signer, err := util.OpenWallet("bidder-client", &cfg.Wallet, chainId)
	if err != nil {
		return nil, errors.Wrap(err, "opening wallet")
	}

	bidValidatorClient, err := rpc.DialContext(ctx, cfg.BidValidatorEndpoint)
	if err != nil {
		return nil, err
	}
	return &BidderClient{
		chainId:                chainId,
		auctionContractAddress: auctionContractAddr,
		client:                 arbClient,
		txOpts:                 txOpts,
		signer:                 signer,
		auctionContract:        auctionContract,
		auctioneerClient:       bidValidatorClient,
		initialRoundTimestamp:  initialTimestamp,
		roundDuration:          roundDuration,
		domainValue:            domainValue,
	}, nil
}

func (bd *BidderClient) Start(ctx_in context.Context) {
	bd.StopWaiter.Start(ctx_in, bd)
}

func (bd *BidderClient) Deposit(ctx context.Context, amount *big.Int) error {
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
	newBid := &Bid{
		ChainId:                bd.chainId,
		ExpressLaneController:  expressLaneController,
		AuctionContractAddress: bd.auctionContractAddress,
		Round:                  CurrentRound(bd.initialRoundTimestamp, bd.roundDuration) + 1,
		Amount:                 amount,
		Signature:              nil,
	}
	sig, err := bd.signer(buildEthereumSignedMessage(newBid.ToMessageBytes()))
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

func buildEthereumSignedMessage(msg []byte) []byte {
	return crypto.Keccak256(append([]byte(fmt.Sprintf("\x19Ethereum Signed Message:\n%d", len(msg))), msg...))
}
