package iostat

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
)

func RegisterAndPopulateMetrics(ctx context.Context, spawnInterval, maxDeviceCount int) {
	if runtime.GOOS != "linux" {
		log.Warn("Iostat command not supported disabling corresponding metrics")
		return
	}
	deviceMetrics := make(map[string]map[string]metrics.GaugeFloat64)
	statReceiver := make(chan DeviceStats)
	go Run(ctx, spawnInterval, statReceiver)
	for {
		stat, ok := <-statReceiver
		if !ok {
			log.Info("Iostat statReceiver channel was closed due to error or command being completed")
			return
		}
		if _, ok := deviceMetrics[stat.DeviceName]; !ok {
			// Register metrics for a maximum of maxDeviceCount (fail safe incase iostat command returns incorrect names indefinitely)
			if len(deviceMetrics) < maxDeviceCount {
				baseMetricName := fmt.Sprintf("isotat/%s/", stat.DeviceName)
				deviceMetrics[stat.DeviceName] = make(map[string]metrics.GaugeFloat64)
				deviceMetrics[stat.DeviceName]["readspersecond"] = metrics.NewRegisteredGaugeFloat64(baseMetricName+"readspersecond", nil)
				deviceMetrics[stat.DeviceName]["writespersecond"] = metrics.NewRegisteredGaugeFloat64(baseMetricName+"writespersecond", nil)
				deviceMetrics[stat.DeviceName]["await"] = metrics.NewRegisteredGaugeFloat64(baseMetricName+"await", nil)
			} else {
				continue
			}
		}
		deviceMetrics[stat.DeviceName]["readspersecond"].Update(stat.ReadsPerSecond)
		deviceMetrics[stat.DeviceName]["writespersecond"].Update(stat.WritesPerSecond)
		deviceMetrics[stat.DeviceName]["await"].Update(stat.Await)
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
	cmd := exec.CommandContext(ctx, "iostat", "-dNxy", strconv.Itoa(interval))
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
	if scanner.Err() != nil {
		log.Error("Iostat scanner error", err, scanner.Err())
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
