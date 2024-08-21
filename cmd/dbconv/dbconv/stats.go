package dbconv

import (
	"sync/atomic"
	"time"
)

type Stats struct {
	entries atomic.Int64
	bytes   atomic.Int64

	startTimestamp       int64
	prevEntries          int64
	prevBytes            int64
	prevEntriesTimestamp int64
	prevBytesTimestamp   int64
}

func (s *Stats) Reset() {
	now := time.Now().UnixNano()
	s.entries.Store(0)
	s.bytes.Store(0)
	s.startTimestamp = now
	s.prevEntries = 0
	s.prevBytes = 0
	s.prevEntriesTimestamp = now
	s.prevBytesTimestamp = now
}

func (s *Stats) LogEntries(entries int64) {
	s.entries.Add(entries)
}

func (s *Stats) Entries() int64 {
	return s.entries.Load()
}

func (s *Stats) LogBytes(bytes int64) {
	s.bytes.Add(bytes)
}

func (s *Stats) Bytes() int64 {
	return s.bytes.Load()
}

func (s *Stats) Elapsed() time.Duration {
	now := time.Now().UnixNano()
	dt := now - s.startTimestamp
	return time.Duration(dt)
}

// not thread safe vs itself
func (s *Stats) EntriesPerSecond() float64 {
	now := time.Now().UnixNano()
	current := s.Entries()
	dt := now - s.prevEntriesTimestamp
	if dt == 0 {
		dt = 1
	}
	de := current - s.prevEntries
	s.prevEntries = current
	s.prevEntriesTimestamp = now
	return float64(de) * 1e9 / float64(dt)
}

// not thread safe vs itself
func (s *Stats) BytesPerSecond() float64 {
	now := time.Now().UnixNano()
	current := s.Bytes()
	dt := now - s.prevBytesTimestamp
	if dt == 0 {
		dt = 1
	}
	db := current - s.prevBytes
	s.prevBytes = current
	s.prevBytesTimestamp = now
	return float64(db) * 1e9 / float64(dt)
}

func (s *Stats) AverageEntriesPerSecond() float64 {
	now := time.Now().UnixNano()
	dt := now - s.startTimestamp
	if dt == 0 {
		dt = 1
	}
	return float64(s.Entries()) * 1e9 / float64(dt)
}

func (s *Stats) AverageBytesPerSecond() float64 {
	now := time.Now().UnixNano()
	dt := now - s.startTimestamp
	if dt == 0 {
		dt = 1
	}
	return float64(s.Bytes()) * 1e9 / float64(dt)
}
