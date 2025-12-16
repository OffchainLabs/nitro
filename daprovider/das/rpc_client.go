// Copyright 2021-2025
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package das

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/blsSignatures"
	"github.com/offchainlabs/nitro/daprovider/das/dasutil"
	"github.com/offchainlabs/nitro/daprovider/data_streaming"
	"github.com/offchainlabs/nitro/util/pretty"
	"github.com/offchainlabs/nitro/util/rpcclient"
	"github.com/offchainlabs/nitro/util/signature"
)

var (
	rpcClientStoreRequestCounter     = metrics.NewRegisteredCounter("arb/das/rpcclient/store/requests", nil)
	rpcClientStoreSuccessCounter     = metrics.NewRegisteredCounter("arb/das/rpcclient/store/success", nil)
	rpcClientStoreFailureCounter     = metrics.NewRegisteredCounter("arb/das/rpcclient/store/failure", nil)
	rpcClientStoreStoredBytesCounter = metrics.NewRegisteredCounter("arb/das/rpcclient/store/bytes", nil)
	rpcClientStoreDurationHistogram  = metrics.NewRegisteredHistogram("arb/das/rpcclient/store/duration", nil, metrics.NewBoundedHistogramSample())
)

// lint:require-exhaustive-initialization
type DASRPCClient struct { // implements DataAvailabilityService
	clnt         *rpc.Client
	url          string
	signer       signature.DataSignerFunc
	dataStreamer *data_streaming.DataStreamer[StoreResult]
}

func nilSigner(_ []byte) ([]byte, error) {
	return []byte{}, nil
}

// lint:require-exhaustive-initialization
type DASRPCClientConfig struct {
	EnableChunkedStore bool                              `koanf:"enable-chunked-store"`
	DataStream         data_streaming.DataStreamerConfig `koanf:"data-stream"`
	RPC                rpcclient.ClientConfig            `koanf:"rpc"`
}

func DASRPCClientConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".enable-chunked-store", true, "enable data to be sent to DAS in chunks instead of all at once")
	data_streaming.DataStreamerConfigAddOptions(prefix+".data-stream", f, DefaultDataStreamRpcMethods)
	rpcclient.RPCClientAddOptions(prefix+".rpc", f, &rpcclient.DefaultClientConfig)
}

var DefaultDataStreamRpcMethods = data_streaming.DataStreamingRPCMethods{
	StartStream:    "das_startChunkedStore",
	StreamChunk:    "das_sendChunk",
	FinalizeStream: "das_commitChunkedStore",
}

func NewDASRPCClient(config *DASRPCClientConfig, signer signature.DataSignerFunc) (*DASRPCClient, error) {
	// Chunked store requires a valid signer for replay protection.
	// The signature is used as a unique request identifier, so nil/empty signatures would cause all requests to be blocked after the first one.
	if config.EnableChunkedStore && signer == nil {
		return nil, errors.New("chunked store requires a valid signer for replay protection; cannot use nil signer")
	}

	if signer == nil {
		signer = nilSigner
	}

	clnt, err := rpc.Dial(config.RPC.URL)
	if err != nil {
		log.Error("Failed to dial DAS RPC server", "url", config.RPC.URL, "err", err)
		return nil, err
	}

	var dataStreamer *data_streaming.DataStreamer[StoreResult]
	if config.EnableChunkedStore {
		payloadSigner := data_streaming.CustomPayloadSigner(func(bytes []byte, extras ...uint64) ([]byte, error) {
			return applyDasSigner(signer, bytes, extras...)
		})

		rpcClient := rpcclient.NewRpcClient(func() *rpcclient.ClientConfig {
			return &config.RPC
		}, nil)
		err := rpcClient.Start(context.Background())
		if err != nil {
			log.Error("Failed to start DAS RPC client", "url", config.RPC.URL, "err", err)
			return nil, err
		}

		dataStreamer, err = data_streaming.NewDataStreamer[StoreResult](config.DataStream, payloadSigner, rpcClient)
		if err != nil {
			log.Error("Failed to create data streamer", "url", config.RPC.URL, "err", err)
			return nil, err
		}
	}

	return &DASRPCClient{
		clnt:         clnt,
		url:          config.RPC.URL,
		signer:       signer,
		dataStreamer: dataStreamer,
	}, nil
}

func (c *DASRPCClient) Store(ctx context.Context, message []byte, timeout uint64) (*dasutil.DataAvailabilityCertificate, error) {
	rpcClientStoreRequestCounter.Inc(1)
	start := time.Now()
	success := false
	defer func() {
		if success {
			rpcClientStoreSuccessCounter.Inc(1)
		} else {
			rpcClientStoreFailureCounter.Inc(1)
		}
		rpcClientStoreDurationHistogram.Update(time.Since(start).Nanoseconds())
	}()

	if c.dataStreamer == nil {
		log.Debug("Legacy store is being force-used by the DAS client", "url", c.url)
		return c.legacyStore(ctx, message, timeout)
	}

	storeResult, err := c.dataStreamer.StreamData(ctx, message, timeout)
	if err != nil {
		if strings.Contains(err.Error(), "the method das_startChunkedStore does not exist") {
			log.Info("Legacy store is used by the DAS client", "url", c.url)
			return c.legacyStore(ctx, message, timeout)
		}
		return nil, err
	}

	respSig, err := blsSignatures.SignatureFromBytes(storeResult.Sig)
	if err != nil {
		return nil, err
	}

	rpcClientStoreStoredBytesCounter.Inc(int64(len(message)))
	success = true

	return &dasutil.DataAvailabilityCertificate{
		DataHash:    common.BytesToHash(storeResult.DataHash),
		Timeout:     uint64(storeResult.Timeout),
		SignersMask: uint64(storeResult.SignersMask),
		Sig:         respSig,
		KeysetHash:  common.BytesToHash(storeResult.KeysetHash),
		Version:     byte(storeResult.Version),
	}, nil
}

func (c *DASRPCClient) legacyStore(ctx context.Context, message []byte, timeout uint64) (*dasutil.DataAvailabilityCertificate, error) {
	// #nosec G115
	log.Trace("das.DASRPCClient.Store(...)", "message", pretty.FirstFewBytes(message), "timeout", time.Unix(int64(timeout), 0), "this", *c)

	reqSig, err := applyDasSigner(c.signer, message, timeout)
	if err != nil {
		return nil, err
	}

	var ret StoreResult
	if err := c.clnt.CallContext(ctx, &ret, "das_store", hexutil.Bytes(message), hexutil.Uint64(timeout), hexutil.Bytes(reqSig)); err != nil {
		return nil, err
	}
	respSig, err := blsSignatures.SignatureFromBytes(ret.Sig)
	if err != nil {
		return nil, err
	}
	return &dasutil.DataAvailabilityCertificate{
		DataHash:    common.BytesToHash(ret.DataHash),
		Timeout:     uint64(ret.Timeout),
		SignersMask: uint64(ret.SignersMask),
		Sig:         respSig,
		KeysetHash:  common.BytesToHash(ret.KeysetHash),
		Version:     byte(ret.Version),
	}, nil
}

func (c *DASRPCClient) String() string {
	return fmt.Sprintf("DASRPCClient{url:%s}", c.url)
}

func (c *DASRPCClient) HealthCheck(ctx context.Context) error {
	return c.clnt.CallContext(ctx, nil, "das_healthCheck")
}

func (c *DASRPCClient) ExpirationPolicy(ctx context.Context) (dasutil.ExpirationPolicy, error) {
	var res string
	err := c.clnt.CallContext(ctx, &res, "das_expirationPolicy")
	if err != nil {
		return -1, err
	}
	return dasutil.StringToExpirationPolicy(res)
}
