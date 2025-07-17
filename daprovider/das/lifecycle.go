// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package das

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/log"
)

type Closer interface {
	Close(ctx context.Context) error
	fmt.Stringer
}

type LifecycleManager struct {
	toClose []Closer
}

func (m *LifecycleManager) Register(c Closer) {
	m.toClose = append(m.toClose, c)
}

func (m *LifecycleManager) StopAndWaitUntil(t time.Duration) {
	if m != nil && m.toClose != nil {
		ctx, cancel := context.WithTimeout(context.Background(), t)
		defer cancel()
		for _, c := range m.toClose {
			err := c.Close(ctx)
			if err != nil {
				log.Warn("Failed to Close DAS component", "err", err)
			}
		}
	}
}
