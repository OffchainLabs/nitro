package dbconv

import (
	"bytes"
	"context"
	"fmt"
	"time"

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

func openDB(config *DBConfig, name string, readonly bool) (ethdb.Database, error) {
	return rawdb.Open(rawdb.OpenOptions{
		Type:               config.DBEngine,
		Directory:          config.Data,
		AncientsDirectory:  "", // don't open freezer
		Namespace:          config.Namespace,
		Cache:              config.Cache,
		Handles:            config.Handles,
		ReadOnly:           readonly,
		PebbleExtraOptions: config.Pebble.ExtraOptions(name),
	})
}

func (c *DBConverter) Convert(ctx context.Context) error {
	var err error
	defer c.Close()
	c.src, err = openDB(&c.config.Src, "src", true)
	if err != nil {
		return err
	}
	c.dst, err = openDB(&c.config.Dst, "dst", false)
	if err != nil {
		return err
	}
	c.stats.Reset()
	it := c.src.NewIterator(nil, nil)
	defer it.Release()
	batch := c.dst.NewBatch()
	entriesInBatch := 0
	for it.Next() && ctx.Err() == nil {
		if err = batch.Put(it.Key(), it.Value()); err != nil {
			return err
		}
		entriesInBatch++
		if batchSize := batch.ValueSize(); batchSize >= c.config.IdealBatchSize {
			if err = batch.Write(); err != nil {
				return err
			}
			c.stats.AddEntries(int64(entriesInBatch))
			c.stats.AddBytes(int64(batchSize))
			batch.Reset()
			entriesInBatch = 0
		}
	}
	if err = ctx.Err(); err == nil {
		batchSize := batch.ValueSize()
		err = batch.Write()
		c.stats.AddEntries(int64(entriesInBatch))
		c.stats.AddBytes(int64(batchSize))
	}
	return err
}

func (c *DBConverter) CompactDestination() error {
	var err error
	c.dst, err = openDB(&c.config.Dst, "dst", false)
	if err != nil {
		return err
	}
	defer c.dst.Close()
	start := time.Now()
	log.Info("Compacting destination database", "data", c.config.Dst.Data)
	if err := c.dst.Compact(nil, nil); err != nil {
		return err
	}
	log.Info("Compaction done", "elapsed", time.Since(start))
	return nil
}

func (c *DBConverter) Verify(ctx context.Context) error {
	if c.config.Verify == 1 {
		log.Info("Starting quick verification - verifying only keys existence")
	} else {
		log.Info("Starting full verification - verifying keys and values")
	}
	var err error
	defer c.Close()
	c.src, err = openDB(&c.config.Src, "src", true)
	if err != nil {
		return err
	}
	c.dst, err = openDB(&c.config.Dst, "dst", true)
	if err != nil {
		return err
	}

	c.stats.Reset()
	it := c.src.NewIterator(nil, nil)
	defer it.Release()
	for it.Next() && ctx.Err() == nil {
		switch c.config.Verify {
		case 1:
			if has, err := c.dst.Has(it.Key()); !has {
				return fmt.Errorf("Missing key in destination db, key: %v, err: %w", it.Key(), err)
			}
			c.stats.AddBytes(int64(len(it.Key())))
		case 2:
			dstValue, err := c.dst.Get(it.Key())
			if err != nil {
				return err
			}
			if !bytes.Equal(dstValue, it.Value()) {
				return fmt.Errorf("Value mismatch for key: %v, src value: %v, dst value: %s", it.Key(), it.Value(), dstValue)
			}
			c.stats.AddBytes(int64(len(it.Key()) + len(dstValue)))
		default:
			return fmt.Errorf("Invalid verify config value: %v", c.config.Verify)
		}
		c.stats.AddEntries(1)
	}
	return ctx.Err()
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
