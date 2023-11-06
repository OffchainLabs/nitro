package eigenda

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/Layr-Labs/eigenda/api/grpc/disperser"
	"github.com/Layr-Labs/eigenda/api/grpc/retriever"
	"github.com/ethereum/go-ethereum/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// EigendaMessageHeaderFlag indicates that this data is a Blob Pointer
// which will be used to retrieve data from EigenDA
const EigendaMessageHeaderFlag byte = 0x0d

func IsEigendaMessageHeaderByte(header byte) bool {
	return (EigendaMessageHeaderFlag & header) > 0
}

type EigenDA struct {
	cfg DAConfig

	// Quorum IDs and SecurityParams to use when dispersing and retrieving blobs
	disperserSecurityParams []*disperser.SecurityParams

	// The total amount of time that the batcher will spend waiting for EigenDA to confirm a blob
	statusQueryTimeout time.Duration

	// The amount of time to wait between status queries of a newly dispersed blob
	statusQueryRetryInterval time.Duration
}

func NewEigenDA(cfg DAConfig) (*EigenDA, error) {
	if cfg.PrimaryAdversaryThreshold == 0 || cfg.PrimaryAdversaryThreshold > 100 {
		return nil, errors.New("invalid primary adversary threshold, must between (0 and 100]")
	}
	if cfg.PrimaryQuorumThreshold == 0 || cfg.PrimaryQuorumThreshold > 100 {
		return nil, errors.New("invalid primary quorum threshold, must between (0 and 100]")
	}

	securityParams := []*disperser.SecurityParams{
		{
			QuorumId:           cfg.PrimaryQuorumID,
			AdversaryThreshold: cfg.PrimaryAdversaryThreshold,
			QuorumThreshold:    cfg.PrimaryQuorumThreshold,
		},
	}

	statusQueryTimeout, err := time.ParseDuration(cfg.StatusQueryTimeout)
	if err != nil {
		return nil, err
	}

	statusQueryRetryInterval, err := time.ParseDuration(cfg.StatusQueryRetryInterval)
	if err != nil {
		return nil, err
	}

	return &EigenDA{
		cfg:                      cfg,
		disperserSecurityParams:  securityParams,
		statusQueryTimeout:       statusQueryTimeout,
		statusQueryRetryInterval: statusQueryRetryInterval,
	}, nil
}

func (c *EigenDA) disperseBlobWithRetry(ctx context.Context, message []byte) (*disperser.BlobStatusReply, error) {
	disperserConn, err := grpc.Dial(c.cfg.DisperserRpc, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	disperserClient := disperser.NewDisperserClient(disperserConn)

	disperseReq := &disperser.DisperseBlobRequest{
		Data:           message,
		SecurityParams: c.disperserSecurityParams,
	}
	disperseRes, err := disperserClient.DisperseBlob(ctx, disperseReq)
	if err != nil || disperseRes == nil {
		log.Error("Unable to disperse blob to EigenDA, aborting", "err", err)
		return nil, err
	}
	if disperseRes.Result == disperser.BlobStatus_UNKNOWN ||
		disperseRes.Result == disperser.BlobStatus_FAILED {
		log.Error("Unable to disperse blob to EigenDA, aborting", "err", err)
		return nil, fmt.Errorf("reply status is %d", disperseRes.Result)
	}

	base64RequestID := base64.StdEncoding.EncodeToString(disperseRes.RequestId)
	log.Info("Blob disepersed to EigenDA, now waiting for confirmation", "requestID", base64RequestID)

	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(ctx, c.statusQueryTimeout)
	defer cancel()

	// Create a ticker that ticks every retryInterval
	ticker := time.NewTicker(c.statusQueryRetryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done(): // Context's deadline is exceeded
			log.Error("EigenDA poll for disperse status retry aborted due to timeout.")
			return nil, errors.New("eigenDA poll for disperse status retry aborted due to timeout")
		case <-ticker.C: // Tick happened, time to try the operation
			statusRes, err := disperserClient.GetBlobStatus(ctx, &disperser.BlobStatusRequest{
				RequestId: disperseRes.RequestId,
			})
			if err != nil {
				log.Warn("Unable to retrieve blob dispersal status, will retry", "requestID", base64RequestID, "err", err)
				continue
			}

			switch statusRes.Status {
			case disperser.BlobStatus_UNKNOWN, disperser.BlobStatus_FAILED:
				log.Error("EigenDA blob dispersal failed in processing", "requestID", base64RequestID, "err", err)
				return nil, fmt.Errorf("eigenDA blob dispersal failed in processing with reply status %d", statusRes.Status)
			case disperser.BlobStatus_PROCESSING:
				log.Warn("Still waiting for confirmation from EigenDA", "requestID", base64RequestID)
			case disperser.BlobStatus_CONFIRMED, disperser.BlobStatus_FINALIZED:
				// TODO(eigenlayer): As long as fault proofs are disabled, we can move on once a blob is confirmed
				// but not yet finalized, without further logic. Once fault proofs are enabled, we will need to update
				// the proposer to wait until the blob associated with an L2 block has been finalized, i.e. the EigenDA
				// contracts on Ethereum have confirmed the full availability of the blob on EigenDA.
				batchHeaderHashHex := fmt.Sprintf("0x%s", hex.EncodeToString(statusRes.Info.BlobVerificationProof.BatchMetadata.BatchHeaderHash))
				log.Info("Successfully dispersed blob to EigenDA", "requestID", base64RequestID, "batchHeaderHash", batchHeaderHashHex)
				return statusRes, nil
			}
		}
	}
}

func (c *EigenDA) Store(ctx context.Context, message []byte) ([]byte, error) {
	statusRes, err := c.disperseBlobWithRetry(ctx, message)
	if err != nil {
		return nil, err
	}
	blobInfo := statusRes.Info

	quorumIDs := make([]uint32, len(blobInfo.BlobHeader.BlobQuorumParams))
	for i := range quorumIDs {
		quorumIDs[i] = blobInfo.BlobHeader.BlobQuorumParams[i].QuorumNumber
	}
	blobRef := &BlobRef{
		BatchHeaderHash:      blobInfo.BlobVerificationProof.BatchMetadata.BatchHeaderHash,
		BlobIndex:            blobInfo.BlobVerificationProof.BlobIndex,
		ReferenceBlockNumber: blobInfo.BlobVerificationProof.BatchMetadata.ConfirmationBlockNumber,
		QuorumIDs:            quorumIDs,
		BlobLength:           uint32(len(message)),
	}

	blobRefData, err := blobRef.MarshalBinary()
	if err != nil {
		log.Warn("BlobRef MashalBinary error", "err", err)
		return nil, err
	}

	buf := new(bytes.Buffer)
	err = binary.Write(buf, binary.BigEndian, EigendaMessageHeaderFlag)
	if err != nil {
		log.Warn("batch type byte serialization failed", "err", err)
		return nil, err
	}

	err = binary.Write(buf, binary.BigEndian, blobRefData)
	if err != nil {
		log.Warn("blob pointer data serialization failed", "err", err)
		return nil, err
	}

	serializedBlobRefData := buf.Bytes()
	log.Info("Succesfully serialized Blob Ref")
	return serializedBlobRefData, nil
}

func (c *EigenDA) Read(blobRef BlobRef) ([]byte, error) {
	log.Info("Requesting data from EigenDA")

	retrieverConn, err := grpc.Dial(c.cfg.RetrieverRpc, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	retrieverClient := retriever.NewRetrieverClient(retrieverConn)

	// TODO: Iterate through quorumIDs if the first one doesn't return.
	blobRequest := &retriever.BlobRequest{
		BatchHeaderHash:      blobRef.BatchHeaderHash,
		BlobIndex:            blobRef.BlobIndex,
		ReferenceBlockNumber: blobRef.ReferenceBlockNumber,
		QuorumId:             blobRef.QuorumIDs[0],
	}
	blob, err := retrieverClient.RetrieveBlob(context.Background(), blobRequest)
	if err != nil {
		return nil, err
	}

	log.Info("Succesfully fetched data from EigenDA")

	return blob.Data[:blobRef.BlobLength], nil
}
