package server_arb

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"runtime/metrics"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/log"
)

type MachineProfilerConfig struct {
	Enable      bool          `koanf:"enable"`
	LogInterval time.Duration `koanf:"log-interval" reload:"hot"`
}

var DefaultMachineProfilerConfig = MachineProfilerConfig{
	Enable:      false,
	LogInterval: 30 * time.Second,
}

func MachineProfilerConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".enable", DefaultMachineProfilerConfig.Enable, "enable arbitrator machine profiling counters and logs")
	f.Duration(prefix+".log-interval", DefaultMachineProfilerConfig.LogInterval, "interval between arbitrator machine profiling logs")
}

type machineProfiler struct {
	enabled atomic.Bool

	machinesCreated            atomic.Int64
	machinesDestroyed          atomic.Int64
	machinesDestroyedFinalizer atomic.Int64
	machinesLive               atomic.Int64
	machineClones              atomic.Int64

	preimageResolvers atomic.Int64
	preimageRefs      atomic.Int64
}

type MachineProfilerSnapshot struct {
	MachinesCreated            int64
	MachinesDestroyed          int64
	MachinesDestroyedFinalizer int64
	MachinesLive               int64
	MachineClones              int64
	PreimageResolvers          int64
	PreimageRefs               int64
}

func newMachineProfiler() *machineProfiler {
	p := &machineProfiler{}
	p.enabled.Store(false)
	return p
}

var globalMachineProfiler = newMachineProfiler()

func (p *machineProfiler) Enabled() bool {
	return p.enabled.Load()
}

func (p *machineProfiler) SetEnabled(enabled bool) {
	p.enabled.Store(enabled)
}

func (p *machineProfiler) OnMachineCreated() {
	if !p.Enabled() {
		return
	}
	p.machinesCreated.Add(1)
	p.adjustLiveCount(1)
}

func (p *machineProfiler) OnMachineDestroyed(fromFinalizer bool) {
	if !p.Enabled() {
		return
	}
	p.machinesDestroyed.Add(1)
	if fromFinalizer {
		p.machinesDestroyedFinalizer.Add(1)
	}
	p.adjustLiveCount(-1)
}

func (p *machineProfiler) OnMachineCloned() {
	if !p.Enabled() {
		return
	}
	p.machineClones.Add(1)
}

func (p *machineProfiler) OnPreimageResolverRegistered() {
	if !p.Enabled() {
		return
	}
	p.preimageResolvers.Add(1)
}

func (p *machineProfiler) OnPreimageResolverReleased() {
	if !p.Enabled() {
		return
	}
	if p.preimageResolvers.Add(-1) < 0 {
		p.preimageResolvers.Store(0)
	}
}

func (p *machineProfiler) OnPreimageRefIncrement() {
	if !p.Enabled() {
		return
	}
	p.preimageRefs.Add(1)
}

func (p *machineProfiler) OnPreimageRefDecrement() {
	if !p.Enabled() {
		return
	}
	if p.preimageRefs.Add(-1) < 0 {
		p.preimageRefs.Store(0)
	}
}

func (p *machineProfiler) adjustLiveCount(delta int64) {
	if p.machinesLive.Add(delta) < 0 {
		p.machinesLive.Store(0)
	}
}

func (p *machineProfiler) Snapshot() MachineProfilerSnapshot {
	return MachineProfilerSnapshot{
		MachinesCreated:            p.machinesCreated.Load(),
		MachinesDestroyed:          p.machinesDestroyed.Load(),
		MachinesDestroyedFinalizer: p.machinesDestroyedFinalizer.Load(),
		MachinesLive:               p.machinesLive.Load(),
		MachineClones:              p.machineClones.Load(),
		PreimageResolvers:          p.preimageResolvers.Load(),
		PreimageRefs:               p.preimageRefs.Load(),
	}
}

func sanitizeProfilerInterval(d time.Duration) time.Duration {
	if d <= 0 {
		d = DefaultMachineProfilerConfig.LogInterval
		if d <= 0 {
			d = time.Minute
		}
	}
	return d
}

func bytesToMiB(b uint64) float64 {
	const reciprocal = 1.0 / (1024.0 * 1024.0)
	return float64(b) * reciprocal
}

func readProcessRSS() (uint64, error) {
	if rss, err := readProcessRSSFromProc(); err == nil {
		return rss, nil
	}
	return readProcessRSSFromMetrics()
}

func readProcessRSSFromProc() (uint64, error) {
	data, err := os.ReadFile("/proc/self/statm")
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

func readProcessRSSFromMetrics() (uint64, error) {
	samples := []metrics.Sample{{Name: "process/resident_memory_bytes"}}
	metrics.Read(samples)
	switch samples[0].Value.Kind() {
	case metrics.KindUint64:
		return samples[0].Value.Uint64(), nil
	case metrics.KindFloat64:
		return uint64(samples[0].Value.Float64()), nil
	default:
		return 0, errors.New("process/resident_memory_bytes metric unavailable")
	}
}

func logProfilerSnapshot(snapshot MachineProfilerSnapshot, native nativeProfilerSnapshot, mem *runtime.MemStats) {
	rssBytes, err := readProcessRSS()
	var rssMiB float64
	if err != nil {
		rssMiB = -1
	} else {
		rssMiB = bytesToMiB(rssBytes)
	}
	log.Info(
		"arbitrator profiler snapshot",
		"machinesLive", snapshot.MachinesLive,
		"machinesCreated", snapshot.MachinesCreated,
		"machinesDestroyed", snapshot.MachinesDestroyed,
		"destroyedByFinalizer", snapshot.MachinesDestroyedFinalizer,
		"machineClones", snapshot.MachineClones,
		"preimageResolvers", snapshot.PreimageResolvers,
		"preimageRefs", snapshot.PreimageRefs,
		"goHeapAllocMB", bytesToMiB(mem.HeapAlloc),
		"goHeapInuseMB", bytesToMiB(mem.HeapInuse),
		"goStackSysMB", bytesToMiB(mem.StackSys),
		"goNumGC", mem.NumGC,
		"goroutines", runtime.NumGoroutine(),
		"rssMB", rssMiB,
		"nativeMachinesLive", native.machinesLive,
		"nativeMachinesCreated", native.machinesCreated,
		"nativeMachinesFreed", native.machinesFreed,
		"nativeMemoryCurrentMB", bytesToMiB(native.memoryCurrentBytes),
		"nativeMemoryPeakMB", bytesToMiB(native.memoryPeakBytes),
		"nativeStylusCurrentMB", bytesToMiB(native.stylusBytesCurrent),
		"nativeStylusPeakMB", bytesToMiB(native.stylusBytesPeak),
		"nativeInboxCurrentMB", bytesToMiB(native.inboxBytesCurrent),
		"nativeInboxEntries", native.inboxEntriesCurrent,
		"nativeLastDestroySteps", native.lastDestroySteps,
		"nativeLastDestroyStatus", native.lastDestroyStatus,
		"nativeLastDestroyMemoryMB", bytesToMiB(native.lastDestroyMemoryBytes),
		"nativeLastDestroyStylusMB", bytesToMiB(native.lastDestroyStylusBytes),
		"nativeLastDestroyInboxMB", bytesToMiB(native.lastDestroyInboxBytes),
	)
}
