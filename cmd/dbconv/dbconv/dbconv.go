package dbconv

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/util/dbutil"
)

type DBConverter struct {
	config *DBConvConfig
	stats  Stats
}

func NewDBConverter(config *DBConvConfig) *DBConverter {
	return &DBConverter{
		config: config,
	}
}

func openDB(config *DBConfig, name string, readonly bool) (ethdb.Database, error) {
	db, err := rawdb.Open(rawdb.OpenOptions{
		Type:      config.DBEngine,
		Directory: config.Data,
		// we don't open freezer, it doesn't need to be converted as it has format independent of db-engine
		// note: user needs to handle copying/moving the ancient directory
		AncientsDirectory:  "",
		Namespace:          config.Namespace,
		Cache:              config.Cache,
		Handles:            config.Handles,
		ReadOnly:           readonly,
		PebbleExtraOptions: config.Pebble.ExtraOptions(name),
	})
	if err != nil {
		return nil, err
	}
	if err := dbutil.UnfinishedConversionCheck(db); err != nil {
		if closeErr := db.Close(); closeErr != nil {
			err = errors.Join(err, closeErr)
		}
		return nil, err
	}

	return db, nil
}

func (c *DBConverter) Convert(ctx context.Context) error {
	var err error
	src, err := openDB(&c.config.Src, "src", true)
	if err != nil {
		return err
	}
	defer src.Close()
	dst, err := openDB(&c.config.Dst, "dst", false)
	if err != nil {
		return err
	}
	defer dst.Close()
	c.stats.Reset()
	log.Info("Converting database", "src", c.config.Src.Data, "dst", c.config.Dst.Data, "db-engine", c.config.Dst.DBEngine)
	if err = dbutil.PutUnfinishedConversionCanary(dst); err != nil {
		return err
	}
	it := src.NewIterator(nil, nil)
	defer it.Release()
	batch := dst.NewBatch()
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
			c.stats.LogEntries(int64(entriesInBatch))
			c.stats.LogBytes(int64(batchSize))
			batch.Reset()
			entriesInBatch = 0
		}
	}
	if err = ctx.Err(); err == nil {
		batchSize := batch.ValueSize()
		if err = batch.Write(); err != nil {
			return err
		}
		c.stats.LogEntries(int64(entriesInBatch))
		c.stats.LogBytes(int64(batchSize))
	}
	if err == nil {
		if err = dbutil.DeleteUnfinishedConversionCanary(dst); err != nil {
			return err
		}
	}
	return err
}

func (c *DBConverter) CompactDestination() error {
	dst, err := openDB(&c.config.Dst, "dst", false)
	if err != nil {
		return err
	}
	defer dst.Close()
	start := time.Now()
	log.Info("Compacting destination database", "dst", c.config.Dst.Data)
	if err := dst.Compact(nil, nil); err != nil {
		return err
	}
	log.Info("Compaction done", "elapsed", time.Since(start))
	return nil
}

func (c *DBConverter) Verify(ctx context.Context) error {
	if c.config.Verify == "keys" {
		log.Info("Starting quick verification - verifying only keys existence")
	} else if c.config.Verify == "full" {
		log.Info("Starting full verification - verifying keys and values")
	}
	var err error
	src, err := openDB(&c.config.Src, "src", true)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := openDB(&c.config.Dst, "dst", true)
	if err != nil {
		return err
	}
	defer dst.Close()

	c.stats.Reset()
	it := src.NewIterator(nil, nil)
	defer it.Release()
	for it.Next() && ctx.Err() == nil {
		switch c.config.Verify {
		case "keys":
			has, err := dst.Has(it.Key())
			if err != nil {
				return fmt.Errorf("Failed to check key existence in destination db, key: %v, err: %w", it.Key(), err)
			}
			if !has {
				return fmt.Errorf("Missing key in destination db, key: %v", it.Key())
			}
			c.stats.LogBytes(int64(len(it.Key())))
		case "full":
			dstValue, err := dst.Get(it.Key())
			if err != nil {
				return err
			}
			if !bytes.Equal(dstValue, it.Value()) {
				return fmt.Errorf("Value mismatch for key: %v, src value: %v, dst value: %s", it.Key(), it.Value(), dstValue)
			}
			c.stats.LogBytes(int64(len(it.Key()) + len(dstValue)))
		default:
			return fmt.Errorf("Invalid verify config value: %v", c.config.Verify)
		}
		c.stats.LogEntries(1)
	}
	return ctx.Err()
}

func (c *DBConverter) Stats() *Stats {
	return &c.stats
}
