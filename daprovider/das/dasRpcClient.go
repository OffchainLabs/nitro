// Copyright 2021-2025
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package das

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/blsSignatures"
	"github.com/offchainlabs/nitro/daprovider/das/dasutil"
	"github.com/offchainlabs/nitro/util/pretty"
	"github.com/offchainlabs/nitro/util/signature"
)

var (
	rpcClientStoreRequestGauge      = metrics.NewRegisteredGauge("arb/das/rpcclient/store/requests", nil)
	rpcClientStoreSuccessGauge      = metrics.NewRegisteredGauge("arb/das/rpcclient/store/success", nil)
	rpcClientStoreFailureGauge      = metrics.NewRegisteredGauge("arb/das/rpcclient/store/failure", nil)
	rpcClientStoreStoredBytesGauge  = metrics.NewRegisteredGauge("arb/das/rpcclient/store/bytes", nil)
	rpcClientStoreDurationHistogram = metrics.NewRegisteredHistogram("arb/das/rpcclient/store/duration", nil, metrics.NewBoundedHistogramSample())

	rpcClientSendChunkSuccessGauge = metrics.NewRegisteredGauge("arb/das/rpcclient/sendchunk/success", nil)
	rpcClientSendChunkFailureGauge = metrics.NewRegisteredGauge("arb/das/rpcclient/sendchunk/failure", nil)
)

type DASRPCClient struct { // implements DataAvailabilityService
	clnt         *rpc.Client
	url          string
	signer       signature.DataSignerFunc
	dataStreamer *DataStreamer
}

func nilSigner(_ []byte) ([]byte, error) {
	return []byte{}, nil
}

func NewDASRPCClient(target string, signer signature.DataSignerFunc, maxStoreChunkBodySize int, enableChunkedStore bool) (*DASRPCClient, error) {
	if signer == nil {
		signer = nilSigner
	}

	clnt, err := rpc.Dial(target)
	if err != nil {
		return nil, err
	}

	var dataStreamer *DataStreamer
	if enableChunkedStore {
		dataStreamer, err = NewDataStreamer(target, maxStoreChunkBodySize, signer)
		if err != nil {
			return nil, err
		}
	}

	return &DASRPCClient{
		clnt:         clnt,
		url:          target,
		signer:       signer,
		dataStreamer: dataStreamer,
	}, nil
}

func (c *DASRPCClient) Store(ctx context.Context, message []byte, timeout uint64) (*dasutil.DataAvailabilityCertificate, error) {
	rpcClientStoreRequestGauge.Inc(1)
	start := time.Now()
	success := false
	defer func() {
		if success {
			rpcClientStoreSuccessGauge.Inc(1)
		} else {
			rpcClientStoreFailureGauge.Inc(1)
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

	rpcClientStoreStoredBytesGauge.Inc(int64(len(message)))
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
