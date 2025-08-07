package generate

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	espresso_client "github.com/EspressoSystems/espresso-network/sdks/go/client"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution"
	execution_engine "github.com/offchainlabs/nitro/system_tests/espresso/execution_engine"
)

// MessageFromSequencerWriter is an interface that defines a method to write
// messages from the sequencer to a writer.
type MessageFromSequencerWriter interface {
	WriteMessageFromSequencer(
		msgIdx arbutil.MessageIndex,
		msgWithMeta arbostypes.MessageWithMetadata,
		msgResult execution.MessageResult,
		blockMetadata common.BlockMetadata,
	) error
}

// Message is a struct that holds the generated message data that
// is needed for the TransactionStreamer to process messages.
type Message struct {
	pos           arbutil.MessageIndex
	msgWithMeta   arbostypes.MessageWithMetadata
	msgResult     execution.MessageResult
	blockMetadata common.BlockMetadata
}

// GenerateMessage is a function that generates a message with a given index
// and a specified size, utilizing the specified hasher to generate the hash.
func GenerateMessage(
	i arbutil.MessageIndex,
	hasher execution_engine.MessageHasher,
	size uint64,
) Message {
	msgData := make([]byte, size)
	_, _ = rand.Read(msgData) // Fill msgData with random bytes
	// We write the index to the message data
	// This can help to identify when debugging issues
	binary.BigEndian.PutUint64(msgData, uint64(i))
	msg := arbostypes.MessageWithMetadataAndBlockInfo{
		MessageWithMeta: arbostypes.MessageWithMetadata{
			Message: &arbostypes.L1IncomingMessage{
				Header: &arbostypes.L1IncomingMessageHeader{
					Kind: arbostypes.L1MessageType_L2Message,
				},
				L2msg: msgData,
			},
		},
	}
	hash := hasher.HashMessageWithMetadata(&msg.MessageWithMeta)
	msgResult := &execution.MessageResult{
		BlockHash: hash,
	}
	return Message{
		pos:           i,
		msgWithMeta:   msg.MessageWithMeta,
		msgResult:     *msgResult,
		blockMetadata: nil,
	}
}

// GenerationBehavior is a function that controls the rate and attributes of
// messages being generated.
type GenerationBehavior func(ctx context.Context, hasher execution_engine.MessageHasher, ch chan<- Message, i arbutil.MessageIndex) arbutil.MessageIndex

// PreloadMessages is a GenerationBehavior that generates N messages without
// delay. The messages are generated with the specified size for convenience.
func PreloadMessages(n arbutil.MessageIndex, size uint64) GenerationBehavior {
	return func(ctx context.Context, hasher execution_engine.MessageHasher, ch chan<- Message, i arbutil.MessageIndex) arbutil.MessageIndex {
		j := arbutil.MessageIndex(0)
		for ; j < n; j++ {
			select {
			default:
			case <-ctx.Done():
				return i + j
			}

			ch <- GenerateMessage(i+j, hasher, size)
		}

		return j + i
	}
}

// ProduceMessagesOfSizeAtInterval is a GenerationBehavior that generates N
// messages of a specific size at a specified interval. This is useful for
// simulating a steady stream of messages being produced over time at
// a consistent interval and time.
func ProduceMessagesOfSizeAtInterval(n arbutil.MessageIndex, size uint64, interval time.Duration) GenerationBehavior {
	return func(ctx context.Context, hasher execution_engine.MessageHasher, ch chan<- Message, i arbutil.MessageIndex) arbutil.MessageIndex {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		j := arbutil.MessageIndex(0)
		for ; j < n; j++ {
			select {
			case <-ctx.Done():
				return i + j

			case <-ticker.C:
				ch <- GenerateMessage(i+j, hasher, size)
			}
		}

		return i + j
	}
}

// DefaultMoltenMessageSize is the default size of the messages generated for
// Molten environments.
//
// This sized is based on the average size of the messages observed in Molten
const DefaultMoltenMessageSize = 3 * 1024 // 3 KiB

// PreloadMoltenMessages is a GenerationBehavior that generates N Molten
// messages without delay.  The messages are generated with the specific
// observed Molten message size.
func PreloadMoltenMessages(n arbutil.MessageIndex) GenerationBehavior {
	return PreloadMessages(n, DefaultMoltenMessageSize)
}

// DefaultMoltenTimeBetweenMessages is the default time interval between
// messages generated for Molten environments.
//
// This duration is based on the average time between messages observed in
// Molten environments.
const DefaultMoltenTimeBetweenMessages = 250 * time.Millisecond

// GenerateStandardMoltenMessages is a GenerationBehavior that generates N
// messages with the specific observed Molten standards.
func GenerateStandardMoltenMessages(n arbutil.MessageIndex) GenerationBehavior {
	return ProduceMessagesOfSizeAtInterval(n, DefaultMoltenMessageSize, 250*time.Millisecond)
}

// ProduceMessages is a function that is meant to be run in a goroutine.
// It continuously generates messages based on the provided behaviors.
// This function will automatically close the channel when all behaviors have
// been executed, ensuring that the channel is properly cleaned up.
func ProduceMessages(ctx context.Context, hasher execution_engine.MessageHasher, ch chan<- Message, behaviors ...GenerationBehavior) {
	defer close(ch)
	var i arbutil.MessageIndex
	for _, behavior := range behaviors {
		select {
		default:
		case <-ctx.Done():
			return
		}

		i = behavior(ctx, hasher, ch, i)
	}
}

// sendMessageToWriter is a helper function that sends a message to the
// MessageFromSequencerWriter. It handles the actual writing of the message
// to the writer, which is typically the TransactionStreamer.
//
// This function simply exists to make the transition between a Message
// and the MessageFromSequencerWriter easier.
func sendMessageToWriter(w MessageFromSequencerWriter, m Message) error {
	return w.WriteMessageFromSequencer(
		m.pos,
		m.msgWithMeta,
		m.msgResult,
		m.blockMetadata,
	)
}

// SendMessageToWriter is a function that is meant to be run in a goroutine.
// It continuously reads messages from the provided channel and sends them to
// the provided MessageFromSequencerWriter. It will return an error if it
// encounters an error while sending a message or if the context is done.
//
// NOTE: This function runs as quickly as possible, so any delay in processing
// depends on the messages coming in from the channel itself.
func SendMessageToWriter(ctx context.Context, w MessageFromSequencerWriter, ch <-chan Message) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case m, ok := <-ch:
			if !ok {
				return nil // Channel closed, exit gracefully
			}

			if err := sendMessageToWriter(w, m); err != nil {
				return err
			}
		}
	}
}

// ConvertEspressoTransactionsInBlockToMessages is a helper function that converts
// Espresso transactions in a block to messages with metadata.
func ConvertEspressoTransactionsInBlockToMessages(
	blockWithTx espresso_client.TransactionsInBlock,
) ([]arbostypes.MessageWithMetadata, error) {
	messagesWithMetadata := make([]arbostypes.MessageWithMetadata, 0, len(blockWithTx.Transactions))

	for _, tx := range blockWithTx.Transactions {
		// We can parse the transactions to get the messages
		// This is a mock function that simulates the parsing of the transaction
		// In a real scenario, this would be replaced with the actual parsing logic
		_, _, _, messages, err := arbutil.ParseHotShotPayload(tx)
		if err != nil {
			return nil, fmt.Errorf("encountered error while parsing transaction: %w", err)
		}

		for _, message := range messages {
			var messageWithMetadata arbostypes.MessageWithMetadata
			if have, want := rlp.DecodeBytes(message, &messageWithMetadata), error(nil); !errors.Is(have, want) {
				return nil, fmt.Errorf("encountered error while decoding message: %w", have)
			}

			messagesWithMetadata = append(messagesWithMetadata, messageWithMetadata)
		}
	}

	return messagesWithMetadata, nil
}
