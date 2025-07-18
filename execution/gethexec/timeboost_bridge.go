package gethexec

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"time"

	// Protobuf imports for grpc calls
	protos "github.com/EspressoSystems/timeboost-proto/go-generated"
	flag "github.com/spf13/pflag"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/ethereum/go-ethereum/arbitrum_types"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/util/stopwaiter"
)

type ForwardService struct {
	protos.UnimplementedForwardApiServer
	processInclusionList func(context.Context, *protos.InclusionList, *arbitrum_types.ConditionalOptions) error
}

// Implement the SubmitInclusionList RPC
func (s *ForwardService) SubmitInclusionList(ctx context.Context, req *protos.InclusionList) (*emptypb.Empty, error) {
	if err := s.processInclusionList(ctx, req, nil); err != nil {
		log.Error("failed to process inclusion list", "err", err)
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

type TimeboostBridge struct {
	stopwaiter.StopWaiter
	config     TimeboostBridgeConfig
	grpcClient protos.InternalApiClient
}

type TimeboostBridgeConfig struct {
	ListenPort               uint16        `koanf:"listen-port"`
	ConnectionTimeout        time.Duration `koanf:"connection-timeout"`
	MaxSendMsgSize           int           `koanf:"max-send-msg-size"`
	MaxReceiveMsgSize        int           `koanf:"max-receive-msg-size"`
	InternalTimeboostGrpcUrl string        `koanf:"internal-timeboost-grpc-url"`
}

var DefaultTimeboostBridgeConfig = TimeboostBridgeConfig{
	ListenPort:               55000,            // Default listen port that timeboost will try and connect to
	ConnectionTimeout:        5 * time.Second,  // Max time for grpc connection timeboost
	MaxSendMsgSize:           5 * 1024 * 1024,  // Max msg receive size from timeboost
	MaxReceiveMsgSize:        5 * 1024 * 1024,  // Max msg send size to timeboost
	InternalTimeboostGrpcUrl: "localhost:5000", // Timeboost grpc server url
}

func TimeboostBridgeConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Uint16(prefix+".listen-port", DefaultTimeboostBridgeConfig.ListenPort, "timeboost inclusion listener listen port")
	f.Duration(prefix+".connection-timeout", DefaultTimeboostBridgeConfig.ConnectionTimeout, "timeboost inclusion list connection timeout")
	f.Int(prefix+".max-send-msg-size", DefaultTimeboostBridgeConfig.MaxSendMsgSize, "timeboost inclusion list send message size")
	f.Int(prefix+".max-receive-msg-size", DefaultTimeboostBridgeConfig.MaxReceiveMsgSize, "timeboost inclusion receive message size")
	f.String(prefix+".internal-timeboost-grpc-url", DefaultTimeboostBridgeConfig.InternalTimeboostGrpcUrl, "timeboost grpc server url")
}

func NewTimeboostBridge(config TimeboostBridgeConfig) (*TimeboostBridge, error) {
	return &TimeboostBridge{
		config:     config,
		grpcClient: nil,
	}, nil
}

// Send block to timeboost who will get certificate over the block hash and forward to hotshot
func (l *TimeboostBridge) SendBlockToTimeboost(block *types.Block, round uint64, chainId uint32) error {
	txns, err := rlp.EncodeToBytes(block.Transactions())
	if err != nil {
		return err
	}
	protoBlock := &protos.Block{
		Namespace: chainId,
		Round:     round,
		Hash:      block.Hash().Bytes(),
		// TODO: Proper hotshot payload
		Payload: txns,
	}
	ctx := context.Background()
	if _, err := l.grpcClient.SubmitBlock(ctx, protoBlock); err != nil {
		log.Error("failed to submit block", "err", err)
		return err
	}
	return nil
}

func (l *TimeboostBridge) Start(
	ctx context.Context,
	processInclusionList func(context.Context, *protos.InclusionList, *arbitrum_types.ConditionalOptions) error,
) error {
	if _, err := url.ParseRequestURI(l.config.InternalTimeboostGrpcUrl); err != nil {
		panic("timeboost grpc url must be a valid url")
	}
	oneMb := 1024 * 1024
	if l.config.MaxSendMsgSize < 5*oneMb || l.config.MaxSendMsgSize > 10*oneMb {
		panic("max send message size should be between 5 and 10 mb")
	}
	if l.config.MaxReceiveMsgSize < 5*oneMb || l.config.MaxReceiveMsgSize > 10*oneMb {
		panic("max receive message size should be bettern 5 and 10 mb")
	}
	if l.config.ConnectionTimeout < 3*time.Second || l.config.ConnectionTimeout > 10*time.Second {
		panic("connection timeout should be between 3 and 10 seconds")
	}

	l.StopWaiter.Start(ctx, l)

	// Grpc connection to timeboost for block submission
	grpcConn, err := grpc.NewClient(l.config.InternalTimeboostGrpcUrl, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Error("Failed to connect to gRPC server", "err", err)
		return err
	}
	l.grpcClient = protos.NewInternalApiClient(grpcConn)

	// Grpc server for inclusion list
	l.LaunchThread(func(ctx context.Context) {
		addr := fmt.Sprintf(":%d", l.config.ListenPort)
		lis, err := net.Listen("tcp", addr)
		if err != nil {
			panic(err)
		}
		server := grpc.NewServer(
			grpc.MaxRecvMsgSize(l.config.MaxSendMsgSize),
			grpc.MaxSendMsgSize(l.config.MaxSendMsgSize),
			grpc.ConnectionTimeout(l.config.ConnectionTimeout),
		)
		protos.RegisterForwardApiServer(server, &ForwardService{
			processInclusionList: processInclusionList,
		})
		go func() {
			<-ctx.Done()
			log.Info("Shutting down gRPC server...")
			server.GracefulStop()
		}()
		if err = server.Serve(lis); err != nil {
			panic(err)
		}
	})
	return nil
}

func (l *TimeboostBridge) StopAndWait() {
	l.StopWaiter.StopAndWait()
}
