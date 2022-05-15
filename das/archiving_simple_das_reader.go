// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"context"
	"errors"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/util/arbmath"
	"sync"
	"time"
)

var ErrArchiveTimeout = errors.New("Archiver timed out")

type ArchivingSimpleDASReader struct {
	inner                  arbstate.SimpleDASReader
	archiveTo              StorageService
	archiveChan            chan archiveItem
	archiveChanClosed      bool
	archiveChanClosedMutex sync.Mutex
	hardStopFunc           func()
	stoppedSignal          chan interface{}
	archiverError          error
}

type archiveItem struct {
	key   []byte
	value []byte
}

func NewArchivingSimpleDASReader(
	ctx context.Context,
	inner arbstate.SimpleDASReader,
	archiveTo StorageService,
	archiveExpirationSeconds uint64,
) (*ArchivingSimpleDASReader, error) {
	archiveChan := make(chan archiveItem, 256)
	hardStopCtx, hardStopFunc := context.WithCancel(ctx)
	ret := &ArchivingSimpleDASReader{
		inner:         inner,
		archiveTo:     archiveTo,
		archiveChan:   archiveChan,
		hardStopFunc:  hardStopFunc,
		stoppedSignal: make(chan interface{}),
	}

	go func() {
		defer close(ret.stoppedSignal)
		for {
			select {
			case item, stillOpen := <-archiveChan:
				if !stillOpen {
					// we successfully archived everything, and our input chan is closed, so shut down cleanly
					return
				}
				expiration := arbmath.SaturatingUAdd(uint64(time.Now().Unix()), archiveExpirationSeconds)
				err := archiveTo.PutByHash(hardStopCtx, item.key, item.value, expiration)
				if err != nil {
					// we hit an error writing to the archive, so record the error and stop archiving
					ret.archiverError = err
					return
				}
			case <-hardStopCtx.Done():
				// hard stop was requested, so terminate early
				ret.archiverError = hardStopCtx.Err()
				return
			}
		}
	}()

	return ret, nil
}

func (r *ArchivingSimpleDASReader) GetByHash(ctx context.Context, hash []byte) ([]byte, error) {
	data, err := r.inner.GetByHash(ctx, hash)
	if err != nil {
		return nil, err
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case r.archiveChan <- archiveItem{hash, data}:
		return data, nil
	}
}

func (r *ArchivingSimpleDASReader) Close(ctx context.Context) error {
	// close archiveChan (if not already closed), so the archiver knows there won't be any more input
	r.archiveChanClosedMutex.Lock()
	if !r.archiveChanClosed {
		close(r.archiveChan)
		r.archiveChanClosed = true
	}
	r.archiveChanClosedMutex.Unlock()

	select {
	case <-ctx.Done():
		// our caller got tired of waiting, so force a hard stop but don't wait for it
		r.hardStopFunc()
		return ctx.Err()
	case <-r.stoppedSignal:
		// archiver finished on its own, so report its error (which is hopefully nil)
		return r.archiverError
	}
}
