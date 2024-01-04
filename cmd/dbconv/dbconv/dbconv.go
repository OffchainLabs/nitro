package dbconv

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
)

type DBConverter struct {
	config *DBConvConfig
	stats  Stats

	src ethdb.Database
	dst ethdb.Database
}

func NewDBConverter(config *DBConvConfig) *DBConverter {
	return &DBConverter{
		config: config,
	}
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

func middleKey(start []byte, end []byte) []byte {
	if len(end) == 0 {
		end = make([]byte, len(start))
		for i := range end {
			end[i] = 0xff
		}
	}
	if len(start) > len(end) {
		tmp := make([]byte, len(start))
		copy(tmp, end)
		end = tmp
	} else if len(start) < len(end) {
		tmp := make([]byte, len(end))
		copy(tmp, start)
		start = tmp
	}
	s := new(big.Int).SetBytes(start)
	e := new(big.Int).SetBytes(end)
	sum := new(big.Int).Add(s, e)
	var m big.Int
	var mid []byte
	if sum.Bit(0) == 1 {
		m.Lsh(sum, 7)
		mid = make([]byte, len(start)+1)
	} else {
		m.Rsh(sum, 1)
		mid = make([]byte, len(start))
	}
	m.FillBytes(mid)
	return mid
}

func (c *DBConverter) copyEntries(ctx context.Context, start []byte, end []byte, wg *sync.WaitGroup, results chan error) {
	log.Debug("copy entries", "start", start, "end", end)
	c.stats.AddThread()
	it := c.src.NewIterator(nil, start)
	defer it.Release()
	var err error
	defer func() {
		results <- err
	}()

	batch := c.dst.NewBatch()
	// TODO support restarting in case of an interruption
	n := 0
	f := 0
	canFork := true
	entriesInBatch := 0
	batchesSinceLastFork := 0
	for it.Next() && ctx.Err() == nil {
		key := it.Key()
		n++
		if len(end) > 0 && bytes.Compare(key, end) >= 0 {
			break
		}
		if err = batch.Put(key, it.Value()); err != nil {
			return
		}
		entriesInBatch++
		if batchSize := batch.ValueSize(); batchSize >= c.config.IdealBatchSize {
			if err = batch.Write(); err != nil {
				return
			}
			c.stats.AddEntries(int64(entriesInBatch))
			c.stats.AddBytes(int64(batchSize))
			entriesInBatch = 0
			batch.Reset()
			batchesSinceLastFork++
		}
		if canFork && batchesSinceLastFork >= c.config.MinBatchesBeforeFork {
			select {
			case err = <-results:
				if err != nil {
					return
				}
				if err = ctx.Err(); err != nil {
					return
				}
				middle := middleKey(key, end)
				if bytes.Compare(middle, key) > 0 && (len(end) == 0 || bytes.Compare(middle, end) < 0) {
					// find next existing key after the middle to prevent the keys from growing too long
					m := c.src.NewIterator(nil, middle)
					if m.Next() {
						foundMiddle := m.Key()
						if len(end) == 0 || bytes.Compare(foundMiddle, end) < 0 {
							wg.Add(1)
							go c.copyEntries(ctx, foundMiddle, end, wg, results)
							middle = foundMiddle
							batchesSinceLastFork = 0
							c.stats.AddFork()
							f++
						} else {
							// no entries either after the middle key or for the middle key
							results <- nil
						}
					} else {
						// no entries either after the middle key or for the middle key
						results <- nil
					}
					end = middle
					m.Release()
				} else {
					log.Warn("no more forking", "key", key, "middle", middle, "end", end)
					canFork = false
					results <- nil
				}
			default:
			}
		}
	}
	if err = ctx.Err(); err == nil {
		batchSize := batch.ValueSize()
		err = batch.Write()
		c.stats.AddEntries(int64(entriesInBatch))
		c.stats.AddBytes(int64(batchSize))
	}
	log.Info("copy entries done", "start", start, "end", end, "n", n, "forked", f)
	c.stats.DecThread()
	wg.Done()
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
	// TODO
	if c.config.Threads <= 0 {
		return errors.New("invalid threads count")
	}

	c.stats.Reset()

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
		c.stats.AddEntries(1)
		c.stats.AddBytes(int64(len(value))) // adding only value len as key is empty
	}
	results := make(chan error, c.config.Threads)
	for i := 0; i < c.config.Threads-1; i++ {
		results <- nil
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go c.copyEntries(ctx, []byte{0}, nil, &wg, results)
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

func (c *DBConverter) Stats() *Stats {
	return &c.stats
}

func (c *DBConverter) Close() {
	if c.src != nil {
		c.src.Close()
	}
	if c.dst != nil {
		c.dst.Close()
	}
}
