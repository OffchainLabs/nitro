package backlog

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	m "github.com/offchainlabs/nitro/broadcaster/message"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/containers"
)

var (
	errDropSegments       = errors.New("remove previous segments from backlog")
	errSequenceNumberSeen = errors.New("sequence number already present in backlog")
	errOutOfBounds        = errors.New("message not found in backlog")

	confirmedSequenceNumberGauge = metrics.NewRegisteredGauge("arb/sequencenumber/confirmed", nil)
	backlogSizeInBytesGauge      = metrics.NewRegisteredGauge("arb/feed/backlog/bytes", nil)
	backlogSizeGauge             = metrics.NewRegisteredGauge("arb/feed/backlog/messages", nil)
)

// Backlog defines the interface for backlog.
type Backlog interface {
	Head() BacklogSegment
	Append(*m.BroadcastMessage) error
	Get(uint64, uint64) (*m.BroadcastMessage, error)
	Count() uint64
	Lookup(uint64) (BacklogSegment, error)
}

// backlog stores backlogSegments and provides the ability to read/write
// messages.
type backlog struct {
	head          atomic.Pointer[backlogSegment]
	tail          atomic.Pointer[backlogSegment]
	lookupByIndex atomic.Pointer[containers.SyncMap[uint64, *backlogSegment]]
	config        ConfigFetcher
	messageCount  atomic.Uint64
}

// NewBacklog creates a backlog.
func NewBacklog(c ConfigFetcher) Backlog {
	b := &backlog{
		config: c,
	}
	b.lookupByIndex.Store(&containers.SyncMap[uint64, *backlogSegment]{})
	return b
}

// Head return the head backlogSegment within the backlog.
func (b *backlog) Head() BacklogSegment {
	return b.head.Load()
}

func (b *backlog) backlogSizeInBytes() (uint64, error) {
	headSeg := b.head.Load()
	tailSeg := b.tail.Load()
	if headSeg == nil || tailSeg == nil {
		if headSeg == nil && tailSeg == nil {
			return 0, nil
		}
		return 0, errors.New("the head or tail segment of feed backlog is nil")
	}

	headSeg.messagesLock.RLock()
	if len(headSeg.messages) == 0 {
		return 0, errors.New("head segment of the feed backlog is empty")
	}
	headMsg := headSeg.messages[0]
	headSeg.messagesLock.RUnlock()

	tailSeg.messagesLock.RLock()
	if len(tailSeg.messages) == 0 {
		return 0, errors.New("tail segment of the feed backlog is empty")
	}
	tailMsg := tailSeg.messages[len(tailSeg.messages)-1]
	size := tailMsg.CumulativeSumMsgSize
	tailSeg.messagesLock.RUnlock()

	size -= headMsg.CumulativeSumMsgSize
	size += headMsg.Size()
	return size, nil
}

// Append will add the given messages to the backlogSegment at head until
// that segment reaches its limit. If messages remain to be added a new segment
// will be created.
func (b *backlog) Append(bm *m.BroadcastMessage) error {

	if bm.ConfirmedSequenceNumberMessage != nil {
		b.delete(uint64(bm.ConfirmedSequenceNumberMessage.SequenceNumber))
		size, err := b.backlogSizeInBytes()
		if err != nil {
			log.Warn("error calculating backlogSizeInBytes", "err", err)
		} else {
			// #nosec G115
			backlogSizeInBytesGauge.Update(int64(size))
		}
	}

	lookupByIndex := b.lookupByIndex.Load()
	for _, msg := range bm.Messages {
		segment := b.tail.Load()
		if segment == nil {
			segment = newBacklogSegment()
			b.head.Store(segment)
			b.tail.Store(segment)
			// #nosec G115
			confirmedSequenceNumberGauge.Update(int64(msg.SequenceNumber))
		}

		prevMsgIdx := segment.End()
		if segment.count() >= b.config().SegmentLimit {
			segment.messagesLock.RLock()
			if len(segment.messages) > 0 {
				msg.CumulativeSumMsgSize = segment.messages[len(segment.messages)-1].CumulativeSumMsgSize
			}
			segment.messagesLock.RUnlock()

			nextSegment := newBacklogSegment()
			segment.nextSegment.Store(nextSegment)
			prevMsgIdx = segment.End()
			segment = nextSegment
			b.tail.Store(segment)
		}

		err := segment.append(prevMsgIdx, msg)
		if errors.Is(err, errDropSegments) {
			head := b.head.Load()
			b.removeFromLookup(head.Start(), uint64(msg.SequenceNumber))
			b.head.Store(segment)
			b.tail.Store(segment)
			b.messageCount.Store(0)
			backlogSizeInBytesGauge.Update(0)
			log.Warn(err.Error())
		} else if errors.Is(err, errSequenceNumberSeen) {
			log.Info("ignoring message sequence number, already in backlog", "message sequence number", msg.SequenceNumber)
			continue
		} else if err != nil {
			return err
		}
		lookupByIndex.Store(uint64(msg.SequenceNumber), segment)
		b.messageCount.Add(1)
		// #nosec G115
		backlogSizeInBytesGauge.Inc(int64(msg.Size()))
	}

	// #nosec G115
	backlogSizeGauge.Update(int64(b.Count()))
	return nil
}

// Get reads messages from the given start to end MessageIndex.
func (b *backlog) Get(start, end uint64) (*m.BroadcastMessage, error) {
	head := b.head.Load()
	tail := b.tail.Load()
	if head == nil && tail == nil {
		return nil, errOutOfBounds
	}

	if end > tail.End() {
		return nil, errOutOfBounds
	}

	segment, err := b.Lookup(start)
	head = b.head.Load()
	if start < head.Start() {
		// doing this check after the Lookup call ensures there is no race
		// condition with a delete call
		start = head.Start()
		segment = head
	} else if err != nil {
		return nil, err
	}

	bm := &m.BroadcastMessage{Version: 1}
	required := end - start + 1
	for {
		segMsgs, err := segment.Get(arbmath.MaxInt(start, segment.Start()), arbmath.MinInt(end, segment.End()))
		if err != nil {
			return nil, err
		}

		bm.Messages = append(bm.Messages, segMsgs...)
		segment = segment.Next()
		if uint64(len(bm.Messages)) == required {
			break
		} else if segment == nil {
			return nil, errOutOfBounds
		}
	}
	return bm, nil
}

// delete removes segments before the confirmed sequence number given. The
// segment containing the confirmed sequence number will continue to store
// previous messages but will register that messages up to the given number
// have been deleted.
func (b *backlog) delete(confirmed uint64) {
	head := b.head.Load()
	tail := b.tail.Load()
	if head == nil && tail == nil {
		return
	}

	if confirmed < head.Start() {
		return
	}

	if confirmed > tail.End() {
		log.Warn("confirmed sequence number is past the end of stored messages", "confirmed sequence number", confirmed, "last stored sequence number", tail.End())
		b.reset()
		return
	}

	// #nosec G115
	confirmedSequenceNumberGauge.Update(int64(confirmed))

	// find the segment containing the confirmed message
	found, err := b.Lookup(confirmed)
	if err != nil {
		log.Error(fmt.Sprintf("%s: clearing backlog", err.Error()))
		b.reset()
		return
	}
	segment, ok := found.(*backlogSegment)
	if !ok {
		log.Error("error in backlogSegment type assertion: clearing backlog")
		b.reset()
		return
	}

	// delete messages from the segment with the confirmed message
	newHead := segment
	start := head.Start()
	if segment.End() == confirmed {
		found = segment.Next()
		newHead, ok = found.(*backlogSegment)
		if !ok {
			log.Error("error in backlogSegment type assertion: clearing backlog")
			b.reset()
			return
		}
	} else {
		err = segment.delete(confirmed)
		if err != nil {
			log.Error(fmt.Sprintf("%s: clearing backlog", err.Error()))
			b.reset()
			return
		}
	}

	// tidy up lookup, count and head
	b.removeFromLookup(start, confirmed)
	count := b.Count() + start - confirmed - uint64(1)
	b.messageCount.Store(count)
	b.head.Store(newHead)
}

// removeFromLookup removes all entries from the head segment's start index to
// the given confirmed index.
func (b *backlog) removeFromLookup(start, end uint64) {
	lookupByIndex := b.lookupByIndex.Load()
	for i := start; i <= end; i++ {
		lookupByIndex.Delete(i)
	}
}

// Lookup attempts to find the backlogSegment storing the given message index.
func (b *backlog) Lookup(i uint64) (BacklogSegment, error) {
	segment, ok := b.lookupByIndex.Load().Load(i)
	if !ok {
		return nil, fmt.Errorf("error finding backlog segment containing message with SequenceNumber %d", i)
	}

	return segment, nil
}

// Count returns the number of messages stored within the backlog.
func (s *backlog) Count() uint64 {
	return s.messageCount.Load()
}

// reset removes all segments from the backlog.
func (b *backlog) reset() {
	b.head.Store(nil)
	b.tail.Store(nil)
	b.lookupByIndex.Store(&containers.SyncMap[uint64, *backlogSegment]{})
	b.messageCount.Store(0)
	backlogSizeInBytesGauge.Update(0)
	backlogSizeGauge.Update(0)
}

// BacklogSegment defines the interface for backlogSegment.
type BacklogSegment interface {
	Start() uint64
	End() uint64
	Next() BacklogSegment
	Contains(uint64) bool
	Messages() []*m.BroadcastFeedMessage
	Get(uint64, uint64) ([]*m.BroadcastFeedMessage, error)
}

// backlogSegment stores messages up to a limit defined by the backlog. It also
// points to the next backlogSegment in the list.
type backlogSegment struct {
	messagesLock sync.RWMutex
	messages     []*m.BroadcastFeedMessage
	nextSegment  atomic.Pointer[backlogSegment]
}

// newBacklogSegment creates a backlogSegment object with an empty slice of
// messages. It does not return an interface as it is only used inside the
// backlog library.
func newBacklogSegment() *backlogSegment {
	return &backlogSegment{
		messages: []*m.BroadcastFeedMessage{},
	}
}

// IsBacklogSegmentNil uses the internal backlogSegment type to check if a
// variable of type BacklogSegment is nil or not. Comparing whether an
// interface is nil directly will not work.
func IsBacklogSegmentNil(segment BacklogSegment) bool {
	if segment == nil {
		return true
	}
	bs, ok := segment.(*backlogSegment)
	if !ok {
		log.Error("error in backlogSegment type assertion: clearing backlog")
		return false
	}
	if bs == nil {
		return true
	}
	return false
}

// Start returns the first message index within the backlogSegment.
func (s *backlogSegment) Start() uint64 {
	s.messagesLock.RLock()
	defer s.messagesLock.RUnlock()
	return s.start()
}

// start allows the first message to be retrieved from functions that already
// have the messagesLock.
func (s *backlogSegment) start() uint64 {
	if len(s.messages) > 0 {
		return uint64(s.messages[0].SequenceNumber)
	}
	return uint64(0)
}

// End returns the last message index within the backlogSegment.
func (s *backlogSegment) End() uint64 {
	s.messagesLock.RLock()
	defer s.messagesLock.RUnlock()
	return s.end()
}

// end allows the first message to be retrieved from functions that already
// have the messagesLock.
func (s *backlogSegment) end() uint64 {
	c := len(s.messages)
	if c == 0 {
		return uint64(0)
	}
	return uint64(s.messages[c-1].SequenceNumber)
}

// Next returns the next backlogSegment.
func (s *backlogSegment) Next() BacklogSegment {
	next := s.nextSegment.Load()
	if next == nil {
		return nil // return a nil interface instead of a nil *backlogSegment
	}
	return next
}

// Messages returns all of the messages stored in the backlogSegment.
func (s *backlogSegment) Messages() []*m.BroadcastFeedMessage {
	s.messagesLock.RLock()
	defer s.messagesLock.RUnlock()
	tmp := make([]*m.BroadcastFeedMessage, len(s.messages))
	copy(tmp, s.messages)
	return tmp
}

// Get reads messages from the given start to end message index.
func (s *backlogSegment) Get(start, end uint64) ([]*m.BroadcastFeedMessage, error) {
	s.messagesLock.RLock()
	defer s.messagesLock.RUnlock()
	noMsgs := []*m.BroadcastFeedMessage{}
	if start < s.start() {
		return noMsgs, errOutOfBounds
	}

	if end > s.end() {
		return noMsgs, errOutOfBounds
	}

	startIndex := start - s.start()
	endIndex := end - s.start() + 1

	tmp := make([]*m.BroadcastFeedMessage, endIndex-startIndex)
	copy(tmp, s.messages[startIndex:endIndex])
	return tmp, nil
}

// append appends the given BroadcastFeedMessage to messages if it is the first
// message in the sequence or the next in the sequence. If segment's end
// message is ahead of the given message append will do nothing. If the given
// message is ahead of the segment's end message append will return
// errDropSegments to ensure any messages before the given message are dropped.
func (s *backlogSegment) append(prevMsgIdx uint64, msg *m.BroadcastFeedMessage) error {
	s.messagesLock.Lock()
	defer s.messagesLock.Unlock()

	prevCumulativeSum := uint64(0)
	if len(s.messages) > 0 {
		prevCumulativeSum = s.messages[len(s.messages)-1].CumulativeSumMsgSize
	}
	if expSeqNum := prevMsgIdx + 1; prevMsgIdx == 0 || uint64(msg.SequenceNumber) == expSeqNum {
		msg.UpdateCumulativeSumMsgSize(prevCumulativeSum)
		s.messages = append(s.messages, msg)
	} else if uint64(msg.SequenceNumber) > expSeqNum {
		msg.UpdateCumulativeSumMsgSize(prevCumulativeSum)
		s.messages = nil
		s.messages = append(s.messages, msg)
		return fmt.Errorf("new message sequence number (%d) is greater than the expected sequence number (%d): %w", msg.SequenceNumber, expSeqNum, errDropSegments)
	} else {
		return errSequenceNumberSeen
	}
	return nil
}

// Contains confirms whether the segment contains a message with the given
// sequence number.
func (s *backlogSegment) Contains(i uint64) bool {
	s.messagesLock.RLock()
	defer s.messagesLock.RUnlock()
	start := s.start()
	if i < start || i > s.end() || len(s.messages) == 0 {
		return false
	}

	msgIndex := i - start
	msg := s.messages[msgIndex]
	return uint64(msg.SequenceNumber) == i
}

// delete removes messages from the backlogSegment up to and including the
// given confirmed message index.
func (s *backlogSegment) delete(confirmed uint64) error {
	start := s.Start()
	end := s.End()
	msgIndex := confirmed - start
	if !s.Contains(confirmed) {
		return fmt.Errorf("confirmed message (%d) is not in expected index (%d) in current backlog (%d-%d)", confirmed, msgIndex, start, end)
	}

	s.messagesLock.Lock()
	s.messages = s.messages[msgIndex+1:]
	s.messagesLock.Unlock()
	return nil
}

// count returns the number of messages stored in the backlog segment.
func (s *backlogSegment) count() int {
	s.messagesLock.RLock()
	defer s.messagesLock.RUnlock()
	return len(s.messages)
}
