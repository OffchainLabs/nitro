package endtoend

import (
	"context"

	"github.com/ethereum/go-ethereum/core/types"

	"github.com/offchainlabs/nitro/bold/testing/endtoend/backend"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

type simpleHeaderProvider struct {
	stopwaiter.StopWaiter
	b   backend.Backend
	chs []chan<- *types.Header
}

func (s *simpleHeaderProvider) Start(ctx context.Context) {
	s.StopWaiter.Start(ctx, s)
	s.LaunchThread(s.listenToHeaders)
}

func (s *simpleHeaderProvider) listenToHeaders(ctx context.Context) {
	ch := make(chan *types.Header, 100)
	sub, err := s.b.Client().SubscribeNewHead(ctx, ch)
	if err != nil {
		panic(err)
	}
	defer sub.Unsubscribe()
	for {
		select {
		case header := <-ch:
			for _, sch := range s.chs {
				sch <- header
			}
		case <-sub.Err():
		case <-ctx.Done():
			return
		}
	}
}

func (s *simpleHeaderProvider) StopAndWait() {
	s.StopWaiter.StopAndWait()
}

func (s *simpleHeaderProvider) Subscribe(requireBlockNrUpdates bool) (<-chan *types.Header, func()) {
	ch := make(chan *types.Header, 100)
	s.chs = append(s.chs, ch)
	return ch, func() {
		s.removeChannel(ch)
		close(ch)
	}
}

func (s *simpleHeaderProvider) removeChannel(ch chan<- *types.Header) {
	for i, sch := range s.chs {
		if sch == ch {
			s.chs = append(s.chs[:i], s.chs[i+1:]...)
			return
		}
	}
}
