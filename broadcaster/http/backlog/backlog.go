package backlog

import (
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbutil"
	m "github.com/offchainlabs/nitro/broadcaster/message"
	"github.com/offchainlabs/nitro/util/arbmath"
)

var (
	errDropSegments       = errors.New("drop previous segments")
	errSequenceNumberSeen = errors.New("sequence number already present in backlog")
	errSequenceOrder      = errors.New("error found in sequence order")
	errOutOfBounds        = errors.New("message not found in backlog")
)

type Backlog struct {
	head          atomic.Pointer[backlogSegment]
	tail          atomic.Pointer[backlogSegment]
	lookupByIndex map[arbutil.MessageIndex]atomic.Pointer[backlogSegment]
	segmentLimit  func() int
	messageCount  atomic.Uint64
}

func NewBacklog(segmentLimit func() int) *Backlog {
	// TODO: add some config stuff
	return &Backlog{segmentLimit: segmentLimit}
}

func (b *Backlog) Get(start, end arbutil.MessageIndex) (*m.BroadcastMessage, error) {
	head := b.head.Load()
	tail := b.tail.Load()
	if head == nil && tail == nil {
		// should this be an empty BM?
		return nil, nil
	}

	if start < head.start {
		return nil, errOutOfBounds
	}

	if end > tail.end {
		return nil, errOutOfBounds
	}

	s, err := b.lookup(start)
	if err != nil {
		return nil, err
	}

	bm := &m.BroadcastMessage{Version: 1}
	for len(bm.Messages) < (int(end-start) + 1) {
		segMsgs, err := s.get(arbmath.MaxInt(start, s.start), arbmath.MinInt(end, s.end))
		if err != nil {
			return nil, err
		}

		bm.Messages = append(bm.Messages, segMsgs...)
		s = s.nextSegment.Load()
		if s == nil {
			return nil, errOutOfBounds
		}
	}
	return bm, nil
}

// Append will add the given messages to the backlogSegment at head until
// that segment reaches its limit. If messages remain to be added a new segment
// will be created.
func (b *Backlog) Append(bm *m.BroadcastMessage) error {

	if bm.ConfirmedSequenceNumberMessage != nil {
		b.delete(bm.ConfirmedSequenceNumberMessage.SequenceNumber)
		// add to metric?
	}

	for _, msg := range bm.Messages {
		s := b.tail.Load()
		if s == nil {
			s = &backlogSegment{}
			b.head.Store(s)
			b.tail.Store(s)
		}

		// check if limit has been reached on segment? perhaps the segment object does not need to know about the limit
		if s.MessageCount() >= b.segmentLimit() {
			nextS := &backlogSegment{}
			s.nextSegment.Store(nextS)
			nextS.previousSegment.Store(s)
			s = nextS
			b.tail.Store(s)
		}

		err := s.append(msg)
		if errors.Is(err, errDropSegments) {
			head := b.head.Load()
			b.removeFromLookup(head.start, msg.SequenceNumber)
			b.head.Store(s)
			b.tail.Store(s)
			b.messageCount.Store(0)
			// remove entries within lookupByIndex up to this latest sequence number
			//b.lookupByIndex = map[uint64]*backlogSegment{msg.SequenceNumber: s}
		} else if errors.Is(err, errSequenceNumberSeen) {
			// message is already in the backlog, do not increase count or add to lookup again
			continue
		} else if err != nil {
			return err
		}
		p := b.lookupByIndex[msg.SequenceNumber]
		p.Store(s)
		b.messageCount.Add(1)
	}

	return nil
}

// delete removes segments before the confirmed sequence number given. It will
// not remove the segment containing the confirmed sequence number.
func (b *Backlog) delete(confirmed arbutil.MessageIndex) {
	// add delete logic

	// if there are no messages then do nothing and return
	head := b.head.Load()
	tail := b.tail.Load()
	if head == nil && tail == nil {
		return
	}

	// if confirmed is lower than first seq number of first seq then do nothing and return
	if confirmed < head.start {
		return
	}

	// if confirmed is greater than end of stored messages remove all messages, potentially log an error, should we return one?
	if confirmed > tail.end {
		log.Error("confirmed sequence number is past the end of stored messages", "confirmed sequence number", confirmed, "last stored sequence number", tail.end)
		b.reset()
		// should this be returning an error? The other buffer does not and just continues
		return
	}

	// weird sequence number found, not expected, drop all messages and log error, might need to be checked in the segment
	s, err := b.lookup(confirmed)
	if err != nil {
		log.Error(fmt.Sprintf("%s: clearing backlog", err.Error()))
		b.reset()
		// should this be returning an error? The other buffer does not and just continues
		return
	}

	if found := s.contains(confirmed); !found {
		log.Error("error message not found in backlog segment, clearing backlog", "confirmed sequence number", confirmed)
		b.reset()
		// should this be returning an error? The other buffer does not and just continues
		return
	}

	// remove segments up to the one that contains the confirmed message
	previous := s.previousSegment.Load()
	if previous == nil {
		return
	}
	b.removeFromLookup(head.start, previous.end)
	b.head.Store(s)
}

// removeFromLookup removes all entries from the head segment's start index to the given confirmed index
func (b *Backlog) removeFromLookup(start arbutil.MessageIndex, end arbutil.MessageIndex) {
	for i := start; i == end; i++ {
		delete(b.lookupByIndex, i)
	}
}

func (b *Backlog) lookup(i arbutil.MessageIndex) (*backlogSegment, error) {
	pointer, ok := b.lookupByIndex[i]
	if !ok {
		return nil, fmt.Errorf("error finding backlog segment containing message with SequenceNumber %d", i)
	}

	s := pointer.Load()
	if s == nil {
		return nil, fmt.Errorf("error loading backlog segment containing message with SequenceNumber %d", i)
	}

	return s, nil
}

func (s *Backlog) MessageCount() int {
	return int(s.messageCount.Load())
}

// reset removes all segments from the backlog
func (b *Backlog) reset() {
	b.head = atomic.Pointer[backlogSegment]{}
	b.tail = atomic.Pointer[backlogSegment]{}
	b.lookupByIndex = map[arbutil.MessageIndex]atomic.Pointer[backlogSegment]{}
}

type backlogSegment struct {
	start           arbutil.MessageIndex
	end             arbutil.MessageIndex
	messages        []*m.BroadcastFeedMessage
	messageCount    atomic.Uint64
	nextSegment     atomic.Pointer[backlogSegment]
	previousSegment atomic.Pointer[backlogSegment]
}

func (s *backlogSegment) get(start, end arbutil.MessageIndex) ([]*m.BroadcastFeedMessage, error) {
	noMsgs := []*m.BroadcastFeedMessage{}
	if start < s.start {
		return noMsgs, errOutOfBounds
	}

	if end > s.end {
		return noMsgs, errOutOfBounds
	}

	startIndex := int(start - s.start)
	endIndex := int(end-s.start) + 1
	return s.messages[startIndex:endIndex], nil
}

// append appends the given BroadcastFeedMessage to messages if it is the first
// message in the sequence or the next in the sequence. If segment's end
// message is ahead of the given message append will do nothing. If the given
// message is ahead of the segment's end message append will return
// errDropSegments to ensure any messages before the given message are dropped.
func (s *backlogSegment) append(msg *m.BroadcastFeedMessage) error {
	defer s.updateSegment()

	if int(s.messageCount.Load()) == 0 {
		s.messages = append(s.messages, msg)
	} else if expSeqNum := s.end + 1; msg.SequenceNumber == expSeqNum {
		s.messages = append(s.messages, msg)
	} else if msg.SequenceNumber > expSeqNum {
		err := fmt.Errorf("message to broadcast has sequence number (%d) greater than the expected sequence number (%d), discarding messages from backlog up to new sequence number: %w", msg.SequenceNumber, expSeqNum, errDropSegments)
		log.Warn(err.Error())
		s.messages = nil
		s.messages = append(s.messages, msg)
		return err
	} else {
		log.Info("skipping already seen message sequence number (%s)", msg.SequenceNumber)
		return errSequenceNumberSeen
	}

	return nil
}

// contains confirms whether the segment contains a message with the given sequence number
func (s *backlogSegment) contains(i arbutil.MessageIndex) bool {
	if i < s.start || i > s.end {
		return false
	}

	msgIndex := uint64(i - s.start)
	msg := s.messages[msgIndex]
	if msg.SequenceNumber == i {
		return true
	}

	return false
	//return false, fmt.Errorf("%w: found sequence number (%d) does not equal expected sequence number (%d)", errSequenceOrder, msg.SequenceNumber, i)
}

// updateSegment updates the messageCount, start and end indices of the segment
// this should be called using defer whenever a method updates the messages
func (s *backlogSegment) updateSegment() {
	c := len(s.messages)
	s.messageCount.Store(uint64(c))
	s.start = s.messages[0].SequenceNumber
	s.end = s.messages[c-1].SequenceNumber
}

func (s *backlogSegment) MessageCount() int {
	return int(s.messageCount.Load())
}
