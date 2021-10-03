package arbstate

import (
	"github.com/ethereum/go-ethereum/rlp"
)

func SplitInboxMessageIntoSegments(message []byte) ([][]byte, error) {
	// TODO
	return [][]byte{message}, nil
}

func DecodeMessageSegment(segment []byte) (ArbMessage, error) {
	// TODO
	var msg ArbMessage
	err := rlp.DecodeBytes(segment, &msg)
	return msg, err
}
