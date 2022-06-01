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
	archiveChan            chan []byte
	archiveChanClosed      bool
	archiveChanClosedMutex sync.Mutex
	hardStopFunc           func()
	stoppedSignal          chan interface{}
	archiverErrorSignal    chan interface{}
	archiverError          error
}

func NewArchivingStorageService(
	ctx context.Context,
	inner StorageService,
	archiveTo StorageService,
	archiveExpirationSeconds uint64,
) (*ArchivingStorageService, error) {
	archiveChan := make(chan []byte, 256)
	hardStopCtx, hardStopFunc := context.WithCancel(ctx)
	ret := &ArchivingStorageService{
		inner:               inner,
		archiveTo:           archiveTo,
		archiveChan:         archiveChan,
		hardStopFunc:        hardStopFunc,
		stoppedSignal:       make(chan interface{}),
		archiverErrorSignal: make(chan interface{}),
	}

	go func() {
		defer close(ret.stoppedSignal)
		anyErrors := false
		for {
			select {
			case data, stillOpen := <-archiveChan:
				if !stillOpen {
					// we successfully archived everything, and our input chan is closed, so shut down cleanly
					return
				}
				expiration := arbmath.SaturatingUAdd(uint64(time.Now().Unix()), archiveExpirationSeconds)
				err := archiveTo.Put(hardStopCtx, data, expiration)
				if err != nil {
					// we hit an error writing to the archive; record the error and keep going
					ret.archiverError = err
					if !anyErrors {
						close(ret.archiverErrorSignal)
						anyErrors = true
					}
				}
			case <-hardStopCtx.Done():
				// hard stop was requested, so terminate early
				ret.archiverError = hardStopCtx.Err()
				if !anyErrors {
					close(ret.archiverErrorSignal)
				}
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
	case serv.archiveChan <- data:
		return data, nil
	}
}

func (serv *ArchivingStorageService) Put(ctx context.Context, data []byte, expiration uint64) error {
	if err := serv.inner.Put(ctx, data, expiration); err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case serv.archiveChan <- data:
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

func (serv *ArchivingStorageService) ExpirationPolicy(ctx context.Context) ExpirationPolicy {
	return DiscardAfterArchiveTimeout
}

func (serv *ArchivingStorageService) GetArchiverErrorSignalChan() <-chan interface{} {
	return serv.archiverErrorSignal
}

func (serv *ArchivingStorageService) GetArchiverError() error {
	return serv.archiverError
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

func (serv *ArchivingSimpleDASReader) GetArchiverErrorSignalChan() <-chan interface{} {
	return serv.wrapped.GetArchiverErrorSignalChan()
}

func (serv *ArchivingSimpleDASReader) GetArchiverError() error {
	return serv.wrapped.GetArchiverError()
}

type limitedStorageService struct {
	arbstate.SimpleDASReader
}

func (lss *limitedStorageService) Put(ctx context.Context, data []byte, expiration uint64) error {
	return errors.New("invalid operation")
}

func (lss *limitedStorageService) Sync(ctx context.Context) error {
	return errors.New("invalid operation")
}

func (lss *limitedStorageService) Close(ctx context.Context) error {
	return errors.New("invalid operation")
}

func (lss *limitedStorageService) ExpirationPolicy(ctx context.Context) ExpirationPolicy {
	return -1
}

func (lss *limitedStorageService) String() string {
	return ""
}
