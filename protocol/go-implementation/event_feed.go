package goimpl

import (
	"context"
	"sync"
)

type EventFeed[Event any] struct {
	mutex    sync.RWMutex
	pingChan chan struct{} // this gets closed and re-generated whenever an event is appended
	ctx      context.Context
	closed   bool
	events   []Event
}

func NewEventFeed[Event any](ctx context.Context) *EventFeed[Event] {
	feed := &EventFeed[Event]{
		mutex:    sync.RWMutex{},
		pingChan: make(chan struct{}),
		ctx:      ctx,
		closed:   false,
		events:   []Event{},
	}
	return feed
}

func (feed *EventFeed[Event]) Append(event Event) {
	feed.mutex.Lock()
	feed.events = append(feed.events, event)
	close(feed.pingChan)
	feed.pingChan = make(chan struct{})
	feed.mutex.Unlock()
}

func (feed *EventFeed[Event]) SubscribeWithFilter(ctx context.Context, c chan<- Event, filter func(Event) bool) {
	go func() {
		defer close(c)
		numRead := 0
		for {
			feed.mutex.RLock()
			closed := feed.closed
			atEnd := false
			var event *Event
			if numRead < len(feed.events) {
				ev := feed.events[numRead]
				if filter(ev) {
					event = &ev
				} else {
					event = nil
				}
				numRead++
			} else {
				atEnd = true
			}
			pingChan := feed.pingChan
			feed.mutex.RUnlock()

			if closed {
				return
			} else if event != nil {
				select {
				case c <- *event:
				case <-ctx.Done():
					return
				case <-feed.ctx.Done():
					return
				}
			} else if atEnd {
				select {
				case <-pingChan:
				case <-ctx.Done():
					return
				case <-feed.ctx.Done():
					return
				}
			} else {
				select {
				case <-ctx.Done():
					return
				case <-feed.ctx.Done():
					return
				default:
				}
			}
		}
	}()
}

func (feed *EventFeed[Event]) Subscribe(ctx context.Context, c chan<- Event) {
	feed.SubscribeWithFilter(ctx, c, func(Event) bool { return true })
}
