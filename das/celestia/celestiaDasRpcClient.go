package celestia

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/rpc"
	celestiaTypes "github.com/offchainlabs/nitro/das/celestia/types"
	"github.com/offchainlabs/nitro/util/pretty"
)

type CelestiaConfig struct {
	Enable bool   `koanf:"enable"`
	URL    string `koanf:"url"`
}

type CelestiaDASClient struct {
	clnt *rpc.Client
	url  string
}

func CelestiaDAConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".enable", false, "Enable Celestia DA")
	f.String(prefix+".url", "http://localhost:9876", "address to use against Celestia DA RPC service")
}

func NewCelestiaDASRPCClient(target string) (*CelestiaDASClient, error) {
	clnt, err := rpc.Dial(target)
	if err != nil {
		return nil, err
	}
	return &CelestiaDASClient{
		clnt: clnt,
		url:  target,
	}, nil
}

func (c *CelestiaDASClient) Store(ctx context.Context, message []byte) ([]byte, error) {
	log.Trace("celestia.CelestiaDASClient.Store(...)", "message", pretty.FirstFewBytes(message))
	ret := []byte{}
	if err := c.clnt.CallContext(ctx, &ret, "celestia_store", hexutil.Bytes(message)); err != nil {
		return nil, err
	}
	log.Info("Got result from Celestia DAS", "result", ret)
	return ret, nil
}

func (c *CelestiaDASClient) String() string {
	return fmt.Sprintf("CelestiaDASClient{url:%s}", c.url)
}

type ReadResult struct {
	Message     []byte     `json:"message"`
	RowRoots    [][]byte   `json:"row_roots"`
	ColumnRoots [][]byte   `json:"column_roots"`
	Rows        [][][]byte `json:"rows"`
	SquareSize  uint64     `json:"square_size"` // Refers to original data square size
	StartRow    uint64     `json:"start_row"`
	EndRow      uint64     `json:"end_row"`
}

func (c *CelestiaDASClient) Read(ctx context.Context, blobPointer *celestiaTypes.BlobPointer) ([]byte, *celestiaTypes.SquareData, error) {
	log.Trace("celestia.CelestiaDASClient.Read(...)", "blobPointer", blobPointer)
	var ret ReadResult
	if err := c.clnt.CallContext(ctx, &ret, "celestia_read", blobPointer); err != nil {
		return nil, nil, err
	}

	squareData := celestiaTypes.SquareData{
		RowRoots:    ret.RowRoots,
		ColumnRoots: ret.ColumnRoots,
		Rows:        ret.Rows,
		SquareSize:  ret.SquareSize,
		StartRow:    ret.StartRow,
		EndRow:      ret.EndRow,
	}

	return ret.Message, &squareData, nil
}

func (c *CelestiaDASClient) GetProof(ctx context.Context, msg []byte) ([]byte, error) {
	res := []byte{}
	err := c.clnt.CallContext(ctx, &res, "celestia_getProof", msg)
	if err != nil {
		return nil, err
	}
	return res, nil
}
