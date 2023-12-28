package dbconv

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
)

type DBConverter struct {
	config *DBConvConfig

	src ethdb.Database
	dst ethdb.Database
}

func NewDBConverter(config *DBConvConfig) *DBConverter {
	return &DBConverter{config: config}
}

func openDB(config *DBConfig, readonly bool) (ethdb.Database, error) {
	return rawdb.Open(rawdb.OpenOptions{
		Type:              config.DBEngine,
		Directory:         config.Data,
		AncientsDirectory: "", // don't open freezer
		Namespace:         "", // TODO do we need metrics namespace?
		Cache:             config.Cache,
		Handles:           config.Handles,
		ReadOnly:          readonly,
	})
}

func (c *DBConverter) copyEntries(ctx context.Context, prefix []byte) error {
	it := c.src.NewIterator(prefix, nil)
	defer it.Release()
	batch := c.dst.NewBatch()
	// TODO support restarting in case of an interruption
	for it.Next() && ctx.Err() == nil {
		key, value := it.Key(), it.Value()
		if err := batch.Put(key, value); err != nil {
			return err
		}
		if batch.ValueSize() >= ethdb.IdealBatchSize {
			if err := batch.Write(); err != nil {
				return err
			}
			batch.Reset()
		}
	}
	if err := batch.Write(); err != nil {
		return err
	}
	return nil
}

func (c *DBConverter) Convert(ctx context.Context) error {
	var err error
	defer c.Close()
	c.src, err = openDB(&c.config.Src, true)
	if err != nil {
		return err
	}
	c.dst, err = openDB(&c.config.Dst, false)
	if err != nil {
		return err
	}
	if c.config.Threads == uint8(0) {
		return errors.New("threads count can't be 0")
	}

	// copy empty key entry
	if has, _ := c.src.Has([]byte{}); has {
		value, err := c.src.Get([]byte{})
		if err != nil {
			return fmt.Errorf("Source database: failed to get value for an empty key: %w", err)
		}
		err = c.dst.Put([]byte{}, value)
		if err != nil {
			return fmt.Errorf("Destination database: failed to put value for an empty key: %w", err)
		}
	}

	results := make(chan error, c.config.Threads)
	for i := uint8(0); i < c.config.Threads; i++ {
		results <- nil
	}
	var wg sync.WaitGroup
	for i := 0; ctx.Err() == nil && i <= 0xff; i++ {
		err = <-results
		if err != nil {
			return err
		}
		prefix := []byte{byte(i)} // TODO make better prefixes, for now we are assuming that majority of keys are 32 byte hashes representing legacy trie nodes
		wg.Add(1)
		go func() {
			results <- c.copyEntries(ctx, prefix)
			wg.Done()
		}()
	}
	wg.Wait()
drainLoop:
	for {
		select {
		case err = <-results:
			if err != nil {
				return err
			}
		default:
			break drainLoop
		}
	}
	return nil
}

func (c *DBConverter) Close() {
	if c.src != nil {
		c.src.Close()
	}
	if c.dst != nil {
		c.dst.Close()
	}
}
