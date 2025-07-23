package chain

import (
	"context"
	"encoding/json"
	"fmt"

	espresso_client "github.com/EspressoSystems/espresso-network/sdks/go/client"
	"github.com/EspressoSystems/espresso-network/sdks/go/types"
	"github.com/EspressoSystems/espresso-network/sdks/go/types/common"
)

// ErrorEspressoClientUnimplementedMethod is an error type that indicates tha
//
//	an Espresso client method is unimplemented.
type ErrorEspressoClientUnimplementedMethod struct {
	Method string
}

// Error implements error
func (e ErrorEspressoClientUnimplementedMethod) Error() string {
	return fmt.Sprintf("unimplemented espresso client method: %s", e.Method)
}

// UnimplementedEspressoClient is an implementation of
// espresso_client.EspressoClient with all methods resulting in a panic
// on invocation.
//
// This is useful, as it allows future implementors to create simpler
// mocks by falling back on this implementation, rather than
// implementing all methods of the interface.
type UnimplementedEspressoClient struct{}

// Compile time check to ensure that UnimplementedEspressoClient implements
// espresso_client.EspressoClient.
var _ espresso_client.EspressoClient = UnimplementedEspressoClient{}

// FetchHeaderByHeight implements client.EspressoClient.
func (UnimplementedEspressoClient) FetchHeaderByHeight(ctx context.Context, height uint64) (types.HeaderImpl, error) {
	panic(ErrorEspressoClientUnimplementedMethod{"FetchHeaderByHeight"})
}

// FetchHeadersByRange implements client.EspressoClient.
func (UnimplementedEspressoClient) FetchHeadersByRange(ctx context.Context, from uint64, until uint64) ([]types.HeaderImpl, error) {
	panic(ErrorEspressoClientUnimplementedMethod{"FetchHeadersByRange"})
}

// FetchLatestBlockHeight implements client.EspressoClient.
func (UnimplementedEspressoClient) FetchLatestBlockHeight(ctx context.Context) (uint64, error) {
	panic(ErrorEspressoClientUnimplementedMethod{"FetchLatestBlockHeight"})
}

// FetchRawHeaderByHeight implements client.EspressoClient.
func (UnimplementedEspressoClient) FetchRawHeaderByHeight(ctx context.Context, height uint64) (json.RawMessage, error) {
	panic(ErrorEspressoClientUnimplementedMethod{"FetchRawHeaderByHeight"})
}

// FetchTransactionByHash implements client.EspressoClient.
func (UnimplementedEspressoClient) FetchTransactionByHash(ctx context.Context, hash *types.TaggedBase64) (types.TransactionQueryData, error) {
	panic(ErrorEspressoClientUnimplementedMethod{"FetchTransactionByHash"})
}

// FetchTransactionsInBlock implements client.EspressoClient.
func (UnimplementedEspressoClient) FetchTransactionsInBlock(ctx context.Context, blockHeight uint64, namespace uint64) (espresso_client.TransactionsInBlock, error) {
	panic(ErrorEspressoClientUnimplementedMethod{"FetchTransactionsInBlock"})
}

// FetchVidCommonByHeight implements client.EspressoClient.
func (UnimplementedEspressoClient) FetchVidCommonByHeight(ctx context.Context, blockHeight uint64) (types.VidCommon, error) {
	panic(ErrorEspressoClientUnimplementedMethod{"FetchVidCommonByHeight"})
}

// SubmitTransaction implements client.EspressoClient.
func (UnimplementedEspressoClient) SubmitTransaction(ctx context.Context, tx common.Transaction) (*common.TaggedBase64, error) {
	panic(ErrorEspressoClientUnimplementedMethod{"SubmitTransaction"})
}

// FetchExplorerTransactionByHash implements client.EspressoClient.
func (UnimplementedEspressoClient) FetchExplorerTransactionByHash(ctx context.Context, hash *types.TaggedBase64) (types.ExplorerTransactionQueryData, error) {
	panic(ErrorEspressoClientUnimplementedMethod{"FetchExplorerTransactionByHash"})
}
