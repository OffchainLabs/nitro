package eigenda

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/Layr-Labs/eigenda/api/grpc/disperser"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbutil"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// EigenDAMessageHeaderFlag indicated that the message is a EigenDARef which will be used to retrieve data from EigenDA
const EigenDAMessageHeaderFlag byte = 0x0c

func IsEigenDAMessageHeaderByte(header byte) bool {
	return (EigenDAMessageHeaderFlag & header) > 0
}

type EigenDAWriter interface {
	Store(context.Context, []byte) (*EigenDARef, error)
	Serialize(eigenDARef *EigenDARef) ([]byte, error)
}

type EigenDAReader interface {
	QueryBlob(ctx context.Context, ref *EigenDARef) ([]byte, error)
}

type EigenDAConfig struct {
	Enable bool   `koanf:"enable"`
	Rpc    string `koanf:"rpc"`
}

func (ec *EigenDAConfig) String() {
	fmt.Println(ec.Enable)
	fmt.Println(ec.Rpc)
	// fmt.Sprintf("enable: %b, rpc: %s", ec.Enable, ec.Rpc)
}

type EigenDARef struct {
	BatchHeaderHash []byte
	BlobIndex       uint32
}

func (b *EigenDARef) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, b.BlobIndex)
	if err != nil {
		return nil, err
	}
	_, err = buf.Write(b.BatchHeaderHash)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (b *EigenDARef) Deserialize(data []byte) error {
	buf := bytes.NewReader(data)
	err := binary.Read(buf, binary.BigEndian, &b.BlobIndex)
	if err != nil {
		return err
	}
	// _, err = buf.Read(b.BatchHeaderHash)
	err = binary.Read(buf, binary.BigEndian, &b.BatchHeaderHash)
	if err != nil {
		return err
	}
	return nil
}

type EigenDA struct {
	client disperser.DisperserClient
}

func NewEigenDA(rpc string) (*EigenDA, error) {
	creds := credentials.NewTLS(&tls.Config{
		InsecureSkipVerify: true,
	})
	conn, err := grpc.Dial(rpc, grpc.WithTransportCredentials(creds))
	if err != nil {
		return nil, err
	}
	return &EigenDA{
		client: disperser.NewDisperserClient(conn),
	}, nil
}

func (e *EigenDA) QueryBlob(ctx context.Context, ref *EigenDARef) ([]byte, error) {
	res, err := e.client.RetrieveBlob(ctx, &disperser.RetrieveBlobRequest{
		BatchHeaderHash: ref.BatchHeaderHash,
		BlobIndex:       ref.BlobIndex,
	})
	if err != nil {
		return nil, err
	}
	return res.GetData(), nil
}

func (e *EigenDA) Store(ctx context.Context, data []byte) (*EigenDARef, error) {
	disperseBlobRequest := &disperser.DisperseBlobRequest{
		Data: data,
		SecurityParams: []*disperser.SecurityParams{
			{QuorumId: 0, AdversaryThreshold: 25, QuorumThreshold: 50},
		},
	}

	res, err := e.client.DisperseBlob(ctx, disperseBlobRequest)
	if err != nil {
		return nil, err
	}

	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()

	var ref *EigenDARef
	for range ticker.C {
		statusReply, err := e.GetBlobStatus(ctx, res.GetRequestId())
		if err != nil {
			log.Error("[eigenda]: GetBlobStatus error: ", err.Error())
			continue
		}
		switch statusReply.GetStatus() {
		case disperser.BlobStatus_CONFIRMED, disperser.BlobStatus_FINALIZED:
			ref = &EigenDARef{
				BatchHeaderHash: statusReply.GetInfo().GetBlobVerificationProof().GetBatchMetadata().GetBatchHeaderHash(),
				BlobIndex:       statusReply.GetInfo().GetBlobVerificationProof().GetBlobIndex(),
			}
			return ref, nil
		case disperser.BlobStatus_FAILED:
			return nil, errors.New("disperser blob failed")
		default:
			continue
		}
	}
	return nil, errors.New("disperser blob query status timeout")

}

func (e *EigenDA) GetBlobStatus(ctx context.Context, reqeustId []byte) (*disperser.BlobStatusReply, error) {
	blockStatusRequest := &disperser.BlobStatusRequest{
		RequestId: reqeustId,
	}
	return e.client.GetBlobStatus(ctx, blockStatusRequest)
}

// Serialize implements EigenDAWriter.
func (e *EigenDA) Serialize(eigenDARef *EigenDARef) ([]byte, error) {
	eigenDARefData, err := eigenDARef.Serialize()
	if err != nil {
		log.Warn("eigenDARef serialize error", "err", err)
		return nil, err
	}
	buf := new(bytes.Buffer)
	err = binary.Write(buf, binary.BigEndian, EigenDAMessageHeaderFlag)
	if err != nil {
		log.Warn("batch type byte serialization failed", "err", err)
		return nil, err
	}
	err = binary.Write(buf, binary.BigEndian, eigenDARefData)

	if err != nil {
		log.Warn("data pointer serialization failed", "err", err)
		return nil, err
	}
	serializedBlobPointerData := buf.Bytes()
	return serializedBlobPointerData, nil
}

func RecoverPayloadFromEigenDABatch(ctx context.Context,
	batchNum uint64,
	sequencerMsg []byte,
	daReader EigenDAReader,
	preimages map[arbutil.PreimageType]map[common.Hash][]byte,
) ([]byte, error) {
	log.Info("Start recovering payload from eigenda: ", "data", hex.EncodeToString(sequencerMsg))
	var shaPreimages map[common.Hash][]byte
	if preimages != nil {
		if preimages[arbutil.Sha2_256PreimageType] == nil {
			preimages[arbutil.Sha2_256PreimageType] = make(map[common.Hash][]byte)
		}
		shaPreimages = preimages[arbutil.Sha2_256PreimageType]
	}
	// 00000020
	// 91c127a758d669ce7c8ed915679653e87bf1dfbcf54d028c522d129c482c897d
	var daRef EigenDARef
	daRef.BlobIndex = binary.BigEndian.Uint32(sequencerMsg[:4])
	daRef.BatchHeaderHash = sequencerMsg[4:]
	log.Info("Data pointer: ", "info", hex.EncodeToString(daRef.BatchHeaderHash), "index", daRef.BlobIndex)
	data, err := daReader.QueryBlob(ctx, &daRef)
	if err != nil {
		log.Error("Failed to query data from EigenDA", "err", err)
		return nil, err
	}
	// log.Info("data: ", "info", hex.EncodeToString(data))
	// record preimage data
	log.Info("Recording preimage data for EigenDA")
	shaDataHash := sha256.New()
	shaDataHash.Write(sequencerMsg)
	dataHash := shaDataHash.Sum([]byte{})
	if shaPreimages != nil {
		shaPreimages[common.BytesToHash(dataHash)] = data
	}
	return data, nil
}
