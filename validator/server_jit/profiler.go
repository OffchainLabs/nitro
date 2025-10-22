package server_jit

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/spf13/pflag"
)

type JitProfilerConfig struct {
	Enable      bool          `koanf:"enable"`
	LogInterval time.Duration `koanf:"log-interval" reload:"hot"`
}

var DefaultJitProfilerConfig = JitProfilerConfig{
	Enable:      false,
	LogInterval: 30 * time.Second,
}

func JitProfilerConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".enable", DefaultJitProfilerConfig.Enable, "enable jit machine profiling counters and logs")
	f.Duration(prefix+".log-interval", DefaultJitProfilerConfig.LogInterval, "interval between jit machine profiling logs")
}

type jitProcessInfo struct {
	pid   int
	start time.Time
}

type jitExitRecord struct {
	pid       int
	errString string
	finished  time.Time
}

type jitProfiler struct {
	enabled atomic.Bool

	machinesStarted atomic.Int64
	machinesExited  atomic.Int64
	machinesLive    atomic.Int64

	activeProcesses sync.Map // pid -> jitProcessInfo
	lastExit        atomic.Value
}

type JitProfilerSnapshot struct {
	MachinesStarted int64
	MachinesExited  int64
	MachinesLive    int64

	ActiveProcesses int
	ActiveRSSBytes  uint64

	LastExitPID int
	LastExitErr string
	LastExitAt  time.Time
}

var globalJitProfiler = newJitProfiler()

func newJitProfiler() *jitProfiler {
	j := &jitProfiler{}
	j.enabled.Store(false)
	j.lastExit.Store(jitExitRecord{})
	return j
}

func (p *jitProfiler) Enabled() bool {
	return p.enabled.Load()
}

func (p *jitProfiler) SetEnabled(enabled bool) {
	p.enabled.Store(enabled)
}

func (p *jitProfiler) OnMachineStarted(pid int) {
	if !p.Enabled() {
		return
	}
	p.machinesStarted.Add(1)
	p.machinesLive.Add(1)
	p.activeProcesses.Store(pid, jitProcessInfo{
		pid:   pid,
		start: time.Now(),
	})
}

func (p *jitProfiler) OnMachineExited(pid int, waitErr error) {
	if !p.Enabled() {
		return
	}
	p.machinesExited.Add(1)
	if p.machinesLive.Add(-1) < 0 {
		p.machinesLive.Store(0)
	}
	p.activeProcesses.Delete(pid)
	record := jitExitRecord{
		pid:      pid,
		finished: time.Now(),
	}
	if waitErr != nil {
		record.errString = waitErr.Error()
	}
	p.lastExit.Store(record)
}

func (p *jitProfiler) Snapshot() JitProfilerSnapshot {
	snapshot := JitProfilerSnapshot{
		MachinesStarted: p.machinesStarted.Load(),
		MachinesExited:  p.machinesExited.Load(),
		MachinesLive:    p.machinesLive.Load(),
	}

	var rssTotal uint64
	active := 0
	p.activeProcesses.Range(func(key, value any) bool {
		pid, ok := key.(int)
		if !ok {
			return true
		}
		active++
		if rss, err := readProcessRSSForPID(pid); err == nil {
			rssTotal += rss
		}
		return true
	})
	snapshot.ActiveProcesses = active
	snapshot.ActiveRSSBytes = rssTotal

	if rec, ok := p.lastExit.Load().(jitExitRecord); ok {
		snapshot.LastExitPID = rec.pid
		snapshot.LastExitErr = rec.errString
		snapshot.LastExitAt = rec.finished
	}
	return snapshot
}

func sanitizeJitProfilerInterval(d time.Duration) time.Duration {
	if d <= 0 {
		d = DefaultJitProfilerConfig.LogInterval
		if d <= 0 {
			d = time.Minute
		}
	}
	return d
}

func readProcessRSSForPID(pid int) (uint64, error) {
	path := fmt.Sprintf("/proc/%d/statm", pid)
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	fields := strings.Fields(string(data))
	if len(fields) < 2 {
		return 0, errors.New("unexpected statm format")
	}
	pages, err := strconv.ParseUint(fields[1], 10, 64)
	if err != nil {
		return 0, err
	}
	pageSize := os.Getpagesize()
	if pageSize <= 0 {
		return 0, fmt.Errorf("unexpected page size %d", pageSize)
	}
	return pages * uint64(pageSize), nil
}

func bytesToMiB(b uint64) float64 {
	const reciprocal = 1.0 / (1024.0 * 1024.0)
	return float64(b) * reciprocal
}

func logJitProfilerSnapshot(snapshot JitProfilerSnapshot, mem *runtime.MemStats) {
	log.Info(
		"jit profiler snapshot",
		"machinesLive", snapshot.MachinesLive,
		"machinesStarted", snapshot.MachinesStarted,
		"machinesExited", snapshot.MachinesExited,
		"activeProcesses", snapshot.ActiveProcesses,
		"activeRSSMB", bytesToMiB(snapshot.ActiveRSSBytes),
		"lastExitPID", snapshot.LastExitPID,
		"lastExitErr", snapshot.LastExitErr,
		"lastExitAt", snapshot.LastExitAt,
		"goHeapAllocMB", bytesToMiB(mem.HeapAlloc),
		"goHeapInuseMB", bytesToMiB(mem.HeapInuse),
		"goStackSysMB", bytesToMiB(mem.StackSys),
		"goNumGC", mem.NumGC,
		"goroutines", runtime.NumGoroutine(),
	)
}
