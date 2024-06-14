package iostat

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
)

type MetricsSpawner struct {
	statReceiver chan DeviceStats
	metrics      map[string]map[string]metrics.GaugeFloat64
}

func NewMetricsSpawner() *MetricsSpawner {
	return &MetricsSpawner{
		metrics:      make(map[string]map[string]metrics.GaugeFloat64),
		statReceiver: make(chan DeviceStats),
	}
}

func (m *MetricsSpawner) RegisterMetrics(ctx context.Context, spwanInterval int) error {
	go Run(ctx, spwanInterval, m.statReceiver)
	// Register metrics for a maximum of 5 devices (fail safe incase iostat command returns incorrect names indefinitely)
	for i := 0; i < 5; i++ {
		stat, ok := <-m.statReceiver
		if !ok {
			return errors.New("failed to register iostat metrics")
		}
		if _, ok := m.metrics[stat.DeviceName]; ok {
			return nil
		}
		baseMetricName := fmt.Sprintf("isotat/%s/", stat.DeviceName)
		m.metrics[stat.DeviceName] = make(map[string]metrics.GaugeFloat64)
		m.metrics[stat.DeviceName]["readspersecond"] = metrics.NewRegisteredGaugeFloat64(baseMetricName+"readspersecond", nil)
		m.metrics[stat.DeviceName]["writespersecond"] = metrics.NewRegisteredGaugeFloat64(baseMetricName+"writespersecond", nil)
		m.metrics[stat.DeviceName]["await"] = metrics.NewRegisteredGaugeFloat64(baseMetricName+"await", nil)
	}
	return nil
}

func (m *MetricsSpawner) PopulateMetrics() {
	for {
		stat, ok := <-m.statReceiver
		if !ok {
			log.Info("Iostat statReceiver channel was closed due to error or command being completed")
			return
		}
		if _, ok := m.metrics[stat.DeviceName]; !ok {
			log.Warn("Unrecognized device name in output of iostat command", "deviceName", stat.DeviceName)
			continue
		}
		m.metrics[stat.DeviceName]["readspersecond"].Update(stat.ReadsPerSecond)
		m.metrics[stat.DeviceName]["writespersecond"].Update(stat.WritesPerSecond)
		m.metrics[stat.DeviceName]["await"].Update(stat.Await)
	}
}

type DeviceStats struct {
	DeviceName      string
	ReadsPerSecond  float64
	WritesPerSecond float64
	Await           float64
}

func Run(ctx context.Context, interval int, receiver chan DeviceStats) {
	defer close(receiver)
	// #nosec G204
	cmd := exec.Command("iostat", "-dNxy", strconv.Itoa(interval))
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Error("Failed to get stdout", "err", err)
		return
	}
	if err := cmd.Start(); err != nil {
		log.Error("Failed to start iostat command", "err", err)
		return
	}
	var fields []string
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		if ctx.Err() != nil {
			log.Error("Context error when running iostat metrics", "err", ctx.Err())
			return
		}
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "Device") {
			fields = strings.Fields(line)
			continue
		}
		data := strings.Fields(line)
		if len(data) == 0 {
			continue
		}
		stat := DeviceStats{}
		var err error
		for i, field := range fields {
			switch field {
			case "Device", "Device:":
				stat.DeviceName = data[i]
			case "r/s":
				stat.ReadsPerSecond, err = strconv.ParseFloat(data[i], 64)
			case "w/s":
				stat.WritesPerSecond, err = strconv.ParseFloat(data[i], 64)
			case "await":
				stat.Await, err = strconv.ParseFloat(data[i], 64)
			}
			if err != nil {
				log.Error("Error parsing command result from iostat", "err", err)
				continue
			}
		}
		if stat.DeviceName == "" {
			continue
		}
		receiver <- stat
	}
	if err := cmd.Process.Kill(); err != nil {
		log.Error("Failed to kill iostat process", "err", err)
	}
	if err := cmd.Wait(); err != nil {
		log.Error("Error waiting for iostat to exit", "err", err)
	}
	stdout.Close()
	log.Info("Iostat command terminated")
}
