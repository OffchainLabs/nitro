package espressostreamer

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"testing"
	"time"

	espressoClient "github.com/EspressoSystems/espresso-network/sdks/go/client"
	"github.com/EspressoSystems/espresso-network/sdks/go/types"
	espressoCommon "github.com/EspressoSystems/espresso-network/sdks/go/types/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/solgen/go/espressogen"
)

func TestEspressoStreamer(t *testing.T) {
	t.Run("Peek should not change the current position", func(t *testing.T) {
		ctx := context.Background()
		mockEspressoClient := new(mockEspressoClient)
		mockEspressoTEEVerifierClient := new(mockEspressoTEEVerifier)

		streamer := NewEspressoStreamer(1, 3, mockEspressoTEEVerifierClient, mockEspressoClient, false, func(l1Height uint64) []common.Address { return []common.Address{} }, 1*time.Second)

		streamer.Reset(1, 3)

		before := streamer.currentMessagePos
		r := streamer.Peek(ctx)
		assert.Nil(t, r)
		assert.Equal(t, before, streamer.currentMessagePos)

		streamer.messageWithMetadataAndPos = []*MessageWithMetadataAndPos{
			{
				MessageWithMeta: arbostypes.MessageWithMetadata{},
				Pos:             1,
				HotshotHeight:   3,
			},
			{
				MessageWithMeta: arbostypes.MessageWithMetadata{},
				Pos:             2,
				HotshotHeight:   4,
			},
		}

		r = streamer.Peek(ctx)
		assert.Equal(t, streamer.messageWithMetadataAndPos[0], r)
		assert.Equal(t, before, streamer.currentMessagePos)
		assert.Equal(t, len(streamer.messageWithMetadataAndPos), 2)
	})
	t.Run("Next should consume a message if it is in buffer", func(t *testing.T) {
		ctx := context.Background()
		mockEspressoClient := new(mockEspressoClient)
		mockEspressoTEEVerifierClient := new(mockEspressoTEEVerifier)

		streamer := NewEspressoStreamer(1, 3, mockEspressoTEEVerifierClient, mockEspressoClient, false, func(l1Height uint64) []common.Address { return []common.Address{} }, 1*time.Second)

		streamer.Reset(1, 3)

		// Empty buffer. Should not change anything
		initialPos := streamer.currentMessagePos
		r := streamer.Next(ctx)
		assert.Nil(t, r)
		assert.Equal(t, initialPos, streamer.currentMessagePos)

		streamer.messageWithMetadataAndPos = []*MessageWithMetadataAndPos{
			{
				MessageWithMeta: arbostypes.MessageWithMetadata{},
				Pos:             1,
				HotshotHeight:   3,
			},
			{
				MessageWithMeta: arbostypes.MessageWithMetadata{},
				Pos:             2,
				HotshotHeight:   4,
			},
		}

		r = streamer.Next(ctx)
		assert.Equal(t, streamer.messageWithMetadataAndPos[0], r)
		assert.Equal(t, initialPos+1, streamer.currentMessagePos)
		// Buffer should still have 2 messages.
		assert.Equal(t, len(streamer.messageWithMetadataAndPos), 2)

		// Second message
		// Peek would cleanup the outdated messages as well
		peekMessage := streamer.Peek(ctx)
		assert.NotNil(t, peekMessage)
		assert.Equal(t, initialPos+1, streamer.currentMessagePos)
		assert.Equal(t, len(streamer.messageWithMetadataAndPos), 1)

		newMessage := streamer.Next(ctx)
		assert.Equal(t, peekMessage, newMessage)
		assert.Equal(t, initialPos+2, streamer.currentMessagePos)

		// Empty message should not alter the current position
		third := streamer.Next(ctx)
		assert.Nil(t, third)
		assert.Equal(t, initialPos+2, streamer.currentMessagePos)
	})
	t.Run("Test should pop messages in order", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		mockEspressoClient := new(mockEspressoClient)
		mockEspressoTEEVerifierClient := new(mockEspressoTEEVerifier)

		// Simulate the call to the tee verifier returning a byte array. To the streamer, this indicates the attestation quote is valid.
		mockEspressoTEEVerifierClient.On("Verify", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(true, nil)
		// create a new streamer object
		streamer := NewEspressoStreamer(1, 1, mockEspressoTEEVerifierClient, mockEspressoClient, false, func(l1Height uint64) []common.Address { return []common.Address{} }, 1*time.Second)
		streamer.Reset(735805, 1)
		// Get the data for this test
		testBlocks := GetTestBlocks()

		mockEspressoClient.On("FetchLatestBlockHeight", ctx).Return(testBlocks[0].blockNumber, nil)
		mockEspressoClient.On("FetchTransactionsInBlock", ctx, testBlocks[0].blockNumber, uint64(1)).Return(testBlocks[0].transactionsInBlock, nil)
		// manually crank the streamers polling function to read an individual hotshot block prepared for the mockEspressoClient
		err := streamer.QueueMessagesFromHotshot(ctx, streamer.parseEspressoTransaction)
		require.NoError(t, err)

		msg := streamer.Next(ctx)
		// Assert that the streamer believe this message to have originated at hotshot height 1
		assert.Equal(t, msg.HotshotHeight, uint64(1))
	})
	t.Run("Streamer should not skip any hotshot blocks", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		mockEspressoClient := new(mockEspressoClient)
		mockEspressoTEEVerifierClient := new(mockEspressoTEEVerifier)

		namespace := uint64(1)
		mockEspressoClient.On("FetchTransactionsInBlock", ctx, uint64(3), namespace).Return(espressoClient.TransactionsInBlock{}, nil)

		mockEspressoClient.On("FetchTransactionsInBlock", ctx, uint64(4), namespace).Return(espressoClient.TransactionsInBlock{}, nil)

		mockEspressoClient.On("FetchTransactionsInBlock", ctx, uint64(5), namespace).Return(espressoClient.TransactionsInBlock{}, nil)

		mockEspressoClient.On("FetchTransactionsInBlock", ctx, uint64(6), namespace).Return(espressoClient.TransactionsInBlock{}, errors.New("test error"))

		streamer := NewEspressoStreamer(namespace, 3, mockEspressoTEEVerifierClient, mockEspressoClient, false, func(l1Height uint64) []common.Address { return []common.Address{} }, 1*time.Second)

		testParseFn := func(tx types.Bytes, l1 uint64) ([]*MessageWithMetadataAndPos, error) {
			return nil, nil
		}

		err := streamer.QueueMessagesFromHotshot(ctx, testParseFn)
		require.NoError(t, err)
		require.Equal(t, streamer.nextHotshotBlockNum, uint64(4))

		err = streamer.QueueMessagesFromHotshot(ctx, testParseFn)
		require.NoError(t, err)
		require.Equal(t, streamer.nextHotshotBlockNum, uint64(5))

		err = streamer.QueueMessagesFromHotshot(ctx, testParseFn)
		require.NoError(t, err)
		require.Equal(t, streamer.nextHotshotBlockNum, uint64(6))

		err = streamer.QueueMessagesFromHotshot(ctx, testParseFn)
		require.Error(t, err)
		require.Equal(t, streamer.nextHotshotBlockNum, uint64(6))

	})
	t.Run("Streamer should query hotshot after being reset", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		mockEspressoClient := new(mockEspressoClient)
		mockEspressoTEEVerifierClient := new(mockEspressoTEEVerifier)

		namespace := uint64(1)
		mockEspressoClient.On("FetchTransactionsInBlock", ctx, uint64(3), namespace).Return(espressoClient.TransactionsInBlock{
			Transactions: []types.Bytes{
				[]byte{0x01, 0x02, 0x03, 0x04},
			},
		}, nil)

		mockEspressoClient.On("FetchTransactionsInBlock", ctx, uint64(4), namespace).Return(espressoClient.TransactionsInBlock{
			Transactions: []types.Bytes{
				[]byte{0x01, 0x02, 0x03, 0x04},
			},
		}, nil)

		streamer := NewEspressoStreamer(namespace, 3, mockEspressoTEEVerifierClient, mockEspressoClient, false, func(l1Height uint64) []common.Address { return []common.Address{} }, 1*time.Second)

		testParseFn := func(pos uint64, hotshotheight uint64) func(tx types.Bytes, l1Height uint64) ([]*MessageWithMetadataAndPos, error) {

			return func(tx types.Bytes, l1Height uint64) ([]*MessageWithMetadataAndPos, error) {
				return []*MessageWithMetadataAndPos{
					{
						MessageWithMeta: arbostypes.MessageWithMetadata{
							Message: &arbostypes.L1IncomingMessage{},
						},
						Pos:           pos,
						HotshotHeight: hotshotheight,
					},
				}, nil
			}
		}

		err := streamer.QueueMessagesFromHotshot(ctx, testParseFn(3, 3))
		require.NoError(t, err)

		err = streamer.QueueMessagesFromHotshot(ctx, testParseFn(4, 4))
		require.NoError(t, err)

		require.Equal(t, 2, len(streamer.messageWithMetadataAndPos))

		streamer.Reset(0, 3)

		require.Equal(t, 0, len(streamer.messageWithMetadataAndPos))

		err = streamer.QueueMessagesFromHotshot(ctx, testParseFn(3, 3))
		require.NoError(t, err)

		require.Equal(t, len(streamer.messageWithMetadataAndPos), 1)
	})

	t.Run("rpc error should retry", func(t *testing.T) {
		ctx := context.Background()
		mockEspressoClient := new(mockEspressoClient)
		namespace := uint64(1)
		blockNum := uint64(3)

		tx1, tx2, tx3 := espressoTypes.Bytes{0x01}, espressoTypes.Bytes{0x02}, espressoTypes.Bytes{0x03}
		mockEspressoClient.On("FetchTransactionsInBlock", ctx, blockNum, namespace).Return(espressoClient.TransactionsInBlock{
			Transactions: []espressoTypes.Bytes{tx1, tx2, tx3},
		}, nil).Once()

		parseAttemptCount := 0
		parseFn := func(tx types.Bytes, _ uint64) ([]*MessageWithMetadataAndPos, error) {
			if assert.ObjectsAreEqual(tx, tx2) {
				parseAttemptCount++
				return nil, rpc.ErrNoResult
			}
			return []*MessageWithMetadataAndPos{{
				MessageWithMeta: arbostypes.MessageWithMetadata{},
				Pos:             uint64(tx[0]),
				HotshotHeight:   blockNum,
			}}, nil
		}

		messages, err := fetchNextHotshotBlock(ctx, mockEspressoClient, blockNum, parseFn, namespace)
		require.NoError(t, err)

		require.Equal(t, 2, len(messages), "Expected to process two messages")
		if len(messages) == 2 {
			assert.Equal(t, uint64(tx1[0]), messages[0].Pos)
			assert.Equal(t, uint64(tx3[0]), messages[1].Pos)
		}

		require.Equal(t, 1, parseAttemptCount, "Expected the failing transaction to be attempted only once")

		mockEspressoClient.AssertExpectations(t)
	})
}

// This serves to assert that we should be expecting a specific error during the test, and if the error does not match, fail the test.
func ExpectErr(t *testing.T, err error, expectedError error) {
	t.Helper()
	if !errors.Is(err, expectedError) {
		t.Fatal(err, expectedError)
	}
}

// This test ensures that parseEspressoTransaction will have
func TestEspressoEmptyTransaction(t *testing.T) {
	mockEspressoClient := new(mockEspressoClient)
	mockEspressoTEEVerifierClient := new(mockEspressoTEEVerifier)
	streamer := NewEspressoStreamer(1, 1, mockEspressoTEEVerifierClient, mockEspressoClient, false, func(l1Height uint64) []common.Address { return []common.Address{} }, time.Millisecond)
	// This determines the contents of the message. For this test the contents of the message needs to be empty (not 0's) to properly test the behavior
	msgFetcher := func(arbutil.MessageIndex) ([]byte, error) {
		return []byte{}, nil
	}
	// create an empty payload
	test := []arbutil.MessageIndex{1, 2}
	payload, _ := arbutil.BuildRawHotShotPayload(test, msgFetcher, 100000) // this value is just a random number to get BuildRawHotShotPayload to return a payload
	// create a fake signature for the payload.
	signerFunc := func([]byte) ([]byte, error) {
		return []byte{1}, nil
	}
	signedPayload, _ := arbutil.SignHotShotPayload(payload, signerFunc)
	_, err := streamer.parseEspressoTransaction(signedPayload, 1)
	ExpectErr(t, err, ErrPayloadHadNoMessages)
}

type TestBlock struct {
	blockNumber         uint64
	transactionsInBlock espressoClient.TransactionsInBlock
}

type mockEspressoTEEVerifier struct {
	mock.Mock
}

func (v *mockEspressoTEEVerifier) Verify(opts *bind.CallOpts, attestation []byte, signature [32]byte) (espressogen.EnclaveReport, error) {
	return espressogen.EnclaveReport{}, nil
}

type mockEspressoClient struct {
	mock.Mock
}

func (m *mockEspressoClient) FetchLatestBlockHeight(ctx context.Context) (uint64, error) {
	args := m.Called(ctx)
	//nolint:errcheck
	return args.Get(0).(uint64), args.Error(1)
}

func (m *mockEspressoClient) FetchExplorerTransactionByHash(ctx context.Context, hash *types.TaggedBase64) (types.ExplorerTransactionQueryData, error) {
	args := m.Called(ctx, hash)
	//nolint:errcheck
	return args.Get(0).(types.ExplorerTransactionQueryData), args.Error(1)
}

func (m *mockEspressoClient) FetchTransactionsInBlock(ctx context.Context, blockHeight uint64, namespace uint64) (espressoClient.TransactionsInBlock, error) {
	args := m.Called(ctx, blockHeight, namespace)
	//nolint:errcheck
	return args.Get(0).(espressoClient.TransactionsInBlock), args.Error(1)
}

func (m *mockEspressoClient) FetchHeaderByHeight(ctx context.Context, blockHeight uint64) (espressoTypes.HeaderImpl, error) {
	header := espressoTypes.Header0_3{Height: blockHeight, L1Finalized: &espressoTypes.L1BlockInfo{Number: 1}}
	return espressoTypes.HeaderImpl{Header: &header}, nil
}

func (m *mockEspressoClient) FetchHeadersByRange(ctx context.Context, from uint64, until uint64) ([]types.HeaderImpl, error) {
	panic("not implemented")
}

func (m *mockEspressoClient) FetchRawHeaderByHeight(ctx context.Context, height uint64) (json.RawMessage, error) {
	panic("not implemented")
}

func (m *mockEspressoClient) FetchTransactionByHash(ctx context.Context, hash *types.TaggedBase64) (types.TransactionQueryData, error) {
	panic("not implemented")
}

func (m *mockEspressoClient) FetchVidCommonByHeight(ctx context.Context, blockHeight uint64) (types.VidCommon, error) {
	panic("not implemented")
}

func (m *mockEspressoClient) SubmitTransaction(ctx context.Context, tx espressoCommon.Transaction) (*espressoCommon.TaggedBase64, error) {
	panic("not implemented")
}

// To generate test scripts for the mock clients, we can create a list of test blocks that we can iterate through and set as the call and return values.
func GetTestBlocks() []TestBlock {
	var data []TestBlock
	hexVal := "000000000000127e03000200000000000b001000939a7233f79c4ca9940a0db3957f0607693b508370c26fea35299d97b79ba551000000000e0e100fffff0100000000000000000001000000000000000000000000000000000000000000000000000000000000000500000000000000e700000000000000c031a57c299ad16d1a4ed0a9a0b658852b9be4037a8229827cdf6634b479f72100000000000000000000000000000000000000000000000000000000000000005fc862cb2e7e1f449f36a18b18aca08c20feaed0d411247816c281d596420cbb000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000098d279ea20f27e8dc4f45ac2036da9af785fa2b1e6f588464b6df471bef885ed0000000000000000000000000000000000000000000000000000000000000000ca100000132c82b9c59153708fc6992a7aa4e47aa74c64b8a2b74ea4852689787953c40ab6377d6094f60a2b48f0b11adc8407783b4633eda899cf5fea08c3d9e5028c8ec7bb42c8900b2530e6b77904411d86468f86d5a5d60f865fc3a09a3da8943986e691bc9065df66bfd9cdd38cbfd0c6ddc107aefa4c7a2c2103fc16334f279e050e0e100fffff0100000000000000000000000000000000000000000000000000000000000000000000000000000000001500000000000000e70000000000000078fe8cfd01095a0f108aff5c40624b93612d6c28b73e1a8d28179c9ddf0e068600000000000000000000000000000000000000000000000000000000000000008c4f5775d796503e96137f77c68a829a0056ac8ded70140b081b094490c57bff00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001000b00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000449f8870a76183a969e56d223e230dd42e5f7a069671f9d0a962bae3b3bc9b590000000000000000000000000000000000000000000000000000000000000000ff82e076a820ecc4d96f1f9031e27ad72a26005a9bc47bb496fad9931fdcc3310b6e4e009a52d31073faef86c4e5043d3c066f80493e3f633c284fd7f6bbb81e2000000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f0500620e00002d2d2d2d2d424547494e2043455254494649434154452d2d2d2d2d0a4d494945386a4343424a6d674177494241674956414f73574367482b6368576e6f6566322f4c4d386b2f466d386e44374d416f4743437147534d343942414d430a4d484178496a416742674e5642414d4d47556c756447567349464e4857434251513073675547786864475a76636d306751304578476a415942674e5642416f4d0a45556c756447567349454e76636e4276636d4630615739754d5251774567594456515148444174545957353059534244624746795954454c4d416b47413155450a4341774351304578437a414a42674e5642415954416c56544d423458445449304d5449794e7a41774d5451314e6c6f5844544d784d5449794e7a41774d5451310a4e6c6f77634445694d434147413155454177775a535735305a5777675530645949464244537942445a584a3061575a70593246305a5445614d426747413155450a43677752535735305a577767513239796347397959585270623234784644415342674e564241634d43314e68626e526849454e7359584a684d517377435159440a5651514944414a445154454c4d416b474131554542684d4356564d775754415442676371686b6a4f5051494242676771686b6a4f50514d4242774e43414153350a744254434b4f532f787852624d71746c4b58445653676445537871617053746c33554b31634772777776544f443477466c61342f4d6d313855773275716d79710a41756c6277782b486d535075642b66774f6352466f344944446a434341776f77487759445652306a42426777466f41556c5739647a62306234656c4153636e550a3944504f4156634c336c5177617759445652306642475177596a42676f46366758495a616148523063484d364c79396863476b7564484a316333526c5a484e6c0a636e5a705932567a4c6d6c75644756734c6d4e766253397a5a3367765932567964476c6d61574e6864476c76626939324d7939775932746a636d772f593245390a6347786864475a76636d306d5a57356a62325270626d63395a4756794d4230474131556444675157424254586437394e595872717075426b385864787245514a0a6d55545a5254414f42674e56485138424166384542414d434273417744415944565230544151482f4241497741444343416a734743537147534962345451454e0a41515343416977776767496f4d42344743697147534962345451454e41514545454939347a796c7647574a7332764f68362f7732553173776767466c42676f710a686b69472b453042445145434d4949425654415142677371686b69472b4530424451454341514942446a415142677371686b69472b45304244514543416749420a446a415142677371686b69472b4530424451454341774942417a415142677371686b69472b4530424451454342414942417a415242677371686b69472b4530420a4451454342514943415038774551594c4b6f5a496876684e41513042416759434167442f4d42414743797147534962345451454e41514948416745424d4241470a43797147534962345451454e41514949416745414d42414743797147534962345451454e4151494a416745414d42414743797147534962345451454e4151494b0a416745414d42414743797147534962345451454e4151494c416745414d42414743797147534962345451454e4151494d416745414d42414743797147534962340a5451454e4151494e416745414d42414743797147534962345451454e4151494f416745414d42414743797147534962345451454e41514950416745414d4241470a43797147534962345451454e41514951416745414d42414743797147534962345451454e415149524167454e4d42384743797147534962345451454e415149530a4242414f44674d442f2f38424141414141414141414141414d42414743697147534962345451454e41514d45416741414d42514743697147534962345451454e0a4151514542674267616741414144415042676f71686b69472b45304244514546436745424d42344743697147534962345451454e415159454549366c773162300a5561597a4f3771345965514a475077775241594b4b6f5a496876684e41513042427a41324d42414743797147534962345451454e415163424151482f4d4241470a43797147534962345451454e41516343415145414d42414743797147534962345451454e41516344415145414d416f4743437147534d343942414d430a4d476778476a415942674e5642414d4d45556c756447567349464e48574342536232393049454e424d526f77474159445651514b4442464a626e526c624342440a62334a7762334a6864476c76626a45554d424947413155454277774c553246756447456751327868636d4578437a414a42674e564241674d416b4e424d5173770a435159445651514745774a56557a4165467730784f4441314d6a45784d4455774d5442614677307a4d7a41314d6a45784d4455774d5442614d484178496a41670a42674e5642414d4d47556c756447567349464e4857434251513073675547786864475a76636d306751304578476a415942674e5642416f4d45556c75644756730a49454e76636e4276636d4630615739754d5251774567594456515148444174545957353059534244624746795954454c4d416b474131554543417743513045780a437a414a42674e5642415954416c56544d466b77457759484b6f5a497a6a3043415159494b6f5a497a6a304441516344516741454e53422f377432316c58534f0a3243757a7078773734654a423732457944476757357258437478327456544c7136684b6b367a2b5569525a436e71523770734f766771466553786c6d546c4a6c0a65546d693257597a33714f42757a43427544416642674e5648534d4547444157674251695a517a575770303069664f44744a5653763141624f536347724442530a42674e5648523845537a424a4d45656752614244686b466f64485277637a6f764c324e6c636e52705a6d6c6a5958526c63793530636e567a6447566b633256790a646d6c6a5a584d75615735305a577775593239744c306c756447567355306459556d397664454e424c6d526c636a416442674e5648513445466751556c5739640a7a62306234656c4153636e553944504f4156634c336c517744675944565230504151482f42415144416745474d42494741315564457745422f7751494d4159420a4166384341514177436759494b6f5a497a6a30454177494452774177524149675873566b6930772b6936565947573355462f32327561586530594a446a3155650a6e412b546a44316169356343494359623153416d4435786b66545670766f34556f79695359787244574c6d5552344349394e4b7966504e2b0a2d2d2d2d2d454e442043455254494649434154452d2d2d2d2d0a2d2d2d2d2d424547494e2043455254494649434154452d2d2d2d2d0a4d4949436c6a4343416a32674177494241674956414a567658633239472b487051456e4a3150517a7a674658433935554d416f4743437147534d343942414d430a4d476778476a415942674e5642414d4d45556c756447567349464e48574342536232393049454e424d526f77474159445651514b4442464a626e526c624342440a62334a7762334a6864476c76626a45554d424947413155454277774c553246756447456751327868636d4578437a414a42674e564241674d416b4e424d5173770a435159445651514745774a56557a4165467730784f4441314d6a45784d4455774d5442614677307a4d7a41314d6a45784d4455774d5442614d484178496a41670a42674e5642414d4d47556c756447567349464e4857434251513073675547786864475a76636d306751304578476a415942674e5642416f4d45556c75644756730a49454e76636e4276636d4630615739754d5251774567594456515148444174545957353059534244624746795954454c4d416b474131554543417743513045780a437a414a42674e5642415954416c56544d466b77457759484b6f5a497a6a3043415159494b6f5a497a6a3044415163445167414543366e45774d4449595a4f6a2f69505773437a61454b69370a314f694f534c52466857476a626e42564a66566e6b59347533496a6b4459594c304d784f346d717379596a6c42616c54565978465032734a424b357a6c4b4f420a757a43427544416642674e5648534d4547444157674251695a517a575770303069664f44744a5653763141624f536347724442530a42674e5648523845537a424a4d45656752614244686b466f64485277637a6f764c324e6c636e52705a6d6c6a5958526c63793530636e567a6447566b633256790a646d6c6a5a584d75615735305a577775593239744c306c756447567355306459556d397664454e424c6d526c636a416442674e564851344546675155496d554d316c71644e496e7a673753560a55723951477a6b6e4271777744675944565230504151482f42415144416745474d42494741315564457745422f7751494d4159424166384341514577436759490a4b6f5a497a6a3045417749445351417752674968414f572f35516b522b533943695344634e6f6f774c7550524c735747662f59693747535839344267775477670a41694541344a306c72486f4d732b586f356f2f7358364f39515778485241765a55474f6452513763767152586171493d0a2d2d2d2d2d454e442043455254494649434154452d2d2d2d2d0a0000000000000b3a3d0000000000000163f90160f90159e10394a4b000000000000000000073657175656e636572837359818467934449c080b9013404f901308305172d8407270e008352d3f8948b14d287b4150ff22ac73df8be720e933f659abc80b8c43161b7f60000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000282e0000000000000000000000000000000000000000000000056bc75e2d63100001000000000000000000000000000000000000000000000000000000001b283e3e000000000000000000000000000000000000000000000000000000000000001084e4c2e4f5a0532b3903b248c357981dbdaddc8e22b137fd23f616ae153eee09ee80451ced37a047d5a4ac9ecd58fccbc8452bfa5e902733391239bdbb8871903c1c20c969c3518301caf300000000000b3a3e0000000000000163f90160f90159e10394a4b000000000000000000073657175656e636572837359818467934449c080b9013404f901308305172e8407270e008352d3f8948b14d287b4150ff22ac73df8be720e933f659abc80b8c43161b7f600000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000002833000000000000000000000000000000000000000000000000002e1a6545b13a8500000000000000000000000000000000000000000000000000000002e9a38636000000000000000000000000000000000000000000000000000000000000001084e4c2e4f5a0a5161a9b1e4299466c59713bcc5f27fe57e5bb1566255dfe17969df310b801cda06944c148f45ac00d07d3b36e72b5bfdb57bfbf85a1d219de984dda93a21aa7228301caf300000000000b3a3f0000000000000163f90160f90159e10394a4b000000000000000000073657175656e636572837359818467934449c080b9013404f901308305172f8407270e008352d3f8948b14d287b4150ff22ac73df8be720e933f659abc80b8c43161b7f600000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000002810000000000000000000000000000000000000000000000000002e13679ef3be6200000000000000000000000000000000000000000000000000000002e9a38636000000000000000000000000000000000000000000000000000000000000001084e4c2e4f6a0e1243adb737aab6847f0d1732b49c024ff02a406e02718077b9ce74eb010a32ca0577ad84a66a43280cfcff304a3effdc961755cb21192e53c255676f0370bf28d8301caf3"
	transactionBytes, err := hex.DecodeString(hexVal)
	if err != nil {
		log.Crit("Failed to decode hex string", "err", err)
	}

	data = append(data, TestBlock{
		blockNumber: 1,
		transactionsInBlock: espressoClient.TransactionsInBlock{
			Transactions: []types.Bytes{transactionBytes},
		},
	})
	return data
}
