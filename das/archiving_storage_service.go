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

type ArchivingStorageService struct {
	inner                  StorageService
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

func NewArchivingStorageService(
	ctx context.Context,
	inner StorageService,
	archiveTo StorageService,
	archiveExpirationSeconds uint64,
) (*ArchivingStorageService, error) {
	archiveChan := make(chan archiveItem, 256)
	hardStopCtx, hardStopFunc := context.WithCancel(ctx)
	ret := &ArchivingStorageService{
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

func (serv *ArchivingStorageService) GetByHash(ctx context.Context, hash []byte) ([]byte, error) {
	data, err := serv.inner.GetByHash(ctx, hash)
	if err != nil {
		return nil, err
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case serv.archiveChan <- archiveItem{hash, data}:
		return data, nil
	}
}

func (serv *ArchivingStorageService) PutByHash(ctx context.Context, hash []byte, data []byte, expiration uint64) error {
	if err := serv.inner.PutByHash(ctx, hash, data, expiration); err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case serv.archiveChan <- archiveItem{hash, data}:
		return nil
	}
}

func (serv *ArchivingStorageService) Sync(ctx context.Context) error { // syncs inner but not the archiver
	return serv.inner.Sync(ctx)
}

func (serv *ArchivingStorageService) Close(ctx context.Context) error {
	// close archiveChan (if not already closed), so the archiver knows there won't be any more input
	serv.archiveChanClosedMutex.Lock()
	if !serv.archiveChanClosed {
		close(serv.archiveChan)
		serv.archiveChanClosed = true
	}
	serv.archiveChanClosedMutex.Unlock()

	select {
	case <-ctx.Done():
		// our caller got tired of waiting, so force a hard stop but don't wait for it
		serv.hardStopFunc()
		return ctx.Err()
	case <-serv.stoppedSignal:
		// archiver finished on its own, so report its error (which is hopefully nil)
		return serv.archiverError
	}
}

func (serv *ArchivingStorageService) String() string {
	return "ArchivingStorageService(" + serv.inner.String() + ")"
}

type ArchivingSimpleDASReader struct {
	wrapped *ArchivingStorageService
}

func NewArchivingSimpleDASReader(
	ctx context.Context,
	inner arbstate.SimpleDASReader,
	archiveTo StorageService,
	archiveExpirationSeconds uint64,
) (*ArchivingSimpleDASReader, error) {
	arch, err := NewArchivingStorageService(ctx, &limitedStorageService{inner}, archiveTo, archiveExpirationSeconds)
	if err != nil {
		return nil, err
	}
	return &ArchivingSimpleDASReader{arch}, nil
}

func (asdr *ArchivingSimpleDASReader) GetByHash(ctx context.Context, hash []byte) ([]byte, error) {
	return asdr.wrapped.GetByHash(ctx, hash)
}

func (asdr *ArchivingSimpleDASReader) Close(ctx context.Context) error {
	return asdr.wrapped.Close(ctx)
}

type limitedStorageService struct {
	arbstate.SimpleDASReader
}

func (lss *limitedStorageService) PutByHash(ctx context.Context, hash []byte, val []byte, expiration uint64) error {
	return errors.New("invalid operation")
}

func (lss *limitedStorageService) Sync(ctx context.Context) error {
	return errors.New("invalid operation")
}

func (lss *limitedStorageService) Close(ctx context.Context) error {
	return errors.New("invalid operation")
}

func (lss *limitedStorageService) String() string {
	return ""
}
