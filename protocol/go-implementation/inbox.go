package goimpl

import (
	"context"
)

type Inbox struct {
	messages [][]byte
	feed     *EventFeed[[]byte]
}

func NewInbox(ctx context.Context) *Inbox {
	return &Inbox{
		messages: [][]byte{},
		feed:     NewEventFeed[[]byte](ctx),
	}
}

func (inbox *Inbox) Subscribe(ctx context.Context, c chan<- []byte) {
	inbox.feed.Subscribe(ctx, c)
}

func (inbox *Inbox) SubscribeWithFilter(ctx context.Context, c chan<- []byte, filter func([]byte) bool) {
	inbox.feed.SubscribeWithFilter(ctx, c, filter)
}

func (inbox *Inbox) Append(tx *ActiveTx, message []byte) {
	tx.verifyReadWrite()
	inbox.messages = append(inbox.messages, message)
	inbox.feed.Append(message)
}

func (inbox *Inbox) NumMessages(tx *ActiveTx) uint64 {
	tx.verifyRead()
	return uint64(len(inbox.messages))
}

func (inbox *Inbox) GetMessage(tx *ActiveTx, num uint64) ([]byte, error) {
	tx.verifyRead()
	if num >= uint64(len(inbox.messages)) {
		return nil, ErrInvalidOp
	}
	return inbox.messages[num], nil
}
