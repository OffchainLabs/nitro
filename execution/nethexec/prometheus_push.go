// Copyright 2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package nethexec

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/prometheus/client_golang/prometheus/push"
	dto "github.com/prometheus/client_model/go"
)

// RegistryGatherer implements prometheus.Gatherer interface for metrics.Registry
type RegistryGatherer struct {
	registry metrics.Registry
	prefix   string
}

// NewRegistryGatherer creates a new Gatherer that wraps a metrics.Registry
func NewRegistryGatherer(registry metrics.Registry, prefix string) *RegistryGatherer {
	return &RegistryGatherer{
		registry: registry,
		prefix:   prefix,
	}
}

// Gather implements prometheus.Gatherer interface
func (rg *RegistryGatherer) Gather() ([]*dto.MetricFamily, error) {
	// Collect all metric names
	var names []string
	rg.registry.Each(func(name string, i interface{}) {
		names = append(names, name)
	})
	sort.Strings(names)

	// Convert each metric to MetricFamily protobuf
	familyMap := make(map[string]*dto.MetricFamily)

	for _, name := range names {
		i := rg.registry.Get(name)
		if err := rg.addMetric(familyMap, name, i); err != nil {
			log.Warn("Failed to convert metric to Prometheus format", "name", name, "error", err)
		}
	}

	// Convert map to slice
	families := make([]*dto.MetricFamily, 0, len(familyMap))
	for _, family := range familyMap {
		families = append(families, family)
	}

	return families, nil
}

// addMetric converts a single metric to Prometheus MetricFamily format
func (rg *RegistryGatherer) addMetric(familyMap map[string]*dto.MetricFamily, name string, i interface{}) error {
	name = rg.applyPrefix(mutateKey(name))

	switch m := i.(type) {
	case *metrics.Counter:
		rg.addGaugeMetric(familyMap, name, float64(m.Snapshot().Count()))
	case *metrics.CounterFloat64:
		rg.addGaugeMetric(familyMap, name, m.Snapshot().Count())
	case *metrics.Gauge:
		rg.addGaugeMetric(familyMap, name, float64(m.Snapshot().Value()))
	case *metrics.GaugeFloat64:
		rg.addGaugeMetric(familyMap, name, m.Snapshot().Value())
	case *metrics.GaugeInfo:
		rg.addGaugeInfoMetric(familyMap, name, m.Snapshot().Value())
	case metrics.Histogram:
		rg.addHistogramMetric(familyMap, name, m.Snapshot())
	case *metrics.Meter:
		rg.addGaugeMetric(familyMap, name, float64(m.Snapshot().Count()))
	case *metrics.Timer:
		rg.addTimerMetric(familyMap, name, m.Snapshot())
	case *metrics.ResettingTimer:
		if m.Snapshot().Count() > 0 {
			rg.addResettingTimerMetric(familyMap, name, m.Snapshot())
		}
	default:
		return fmt.Errorf("unknown metric type %T", i)
	}
	return nil
}

// addGaugeMetric adds a gauge metric to the family map
func (rg *RegistryGatherer) addGaugeMetric(familyMap map[string]*dto.MetricFamily, name string, value float64) {
	metricType := dto.MetricType_GAUGE
	family := rg.getOrCreateFamily(familyMap, name, metricType)

	metric := &dto.Metric{
		Gauge: &dto.Gauge{
			Value: &value,
		},
	}
	family.Metric = append(family.Metric, metric)
}

// addGaugeInfoMetric adds a gauge info metric to the family map
func (rg *RegistryGatherer) addGaugeInfoMetric(familyMap map[string]*dto.MetricFamily, name string, value metrics.GaugeInfoValue) {
	metricType := dto.MetricType_GAUGE
	family := rg.getOrCreateFamily(familyMap, name, metricType)

	labels := make([]*dto.LabelPair, 0, len(value))
	for k, v := range value {
		labelName := k
		labelValue := v
		labels = append(labels, &dto.LabelPair{
			Name:  &labelName,
			Value: &labelValue,
		})
	}
	sort.Slice(labels, func(i, j int) bool {
		return *labels[i].Name < *labels[j].Name
	})

	gaugeValue := float64(1)
	metric := &dto.Metric{
		Label: labels,
		Gauge: &dto.Gauge{
			Value: &gaugeValue,
		},
	}
	family.Metric = append(family.Metric, metric)
}

// addHistogramMetric adds a histogram metric as a summary to the family map
func (rg *RegistryGatherer) addHistogramMetric(familyMap map[string]*dto.MetricFamily, name string, h metrics.HistogramSnapshot) {
	metricType := dto.MetricType_SUMMARY
	family := rg.getOrCreateFamily(familyMap, name, metricType)

	pv := []float64{0.5, 0.75, 0.95, 0.99, 0.999, 0.9999}
	ps := h.Percentiles(pv)

	quantiles := make([]*dto.Quantile, len(pv))
	for i := range pv {
		quantile := pv[i]
		value := ps[i]
		quantiles[i] = &dto.Quantile{
			Quantile: &quantile,
			Value:    &value,
		}
	}

	count := uint64(h.Count())
	metric := &dto.Metric{
		Summary: &dto.Summary{
			SampleCount: &count,
			Quantile:    quantiles,
		},
	}
	family.Metric = append(family.Metric, metric)
}

// addTimerMetric adds a timer metric as a summary to the family map
func (rg *RegistryGatherer) addTimerMetric(familyMap map[string]*dto.MetricFamily, name string, t *metrics.TimerSnapshot) {
	metricType := dto.MetricType_SUMMARY
	family := rg.getOrCreateFamily(familyMap, name, metricType)

	pv := []float64{0.5, 0.75, 0.95, 0.99, 0.999, 0.9999}
	ps := t.Percentiles(pv)

	quantiles := make([]*dto.Quantile, len(pv))
	for i := range pv {
		quantile := pv[i]
		value := ps[i]
		quantiles[i] = &dto.Quantile{
			Quantile: &quantile,
			Value:    &value,
		}
	}

	count := uint64(t.Count())
	metric := &dto.Metric{
		Summary: &dto.Summary{
			SampleCount: &count,
			Quantile:    quantiles,
		},
	}
	family.Metric = append(family.Metric, metric)
}

// addResettingTimerMetric adds a resetting timer metric as a summary to the family map
func (rg *RegistryGatherer) addResettingTimerMetric(familyMap map[string]*dto.MetricFamily, name string, t *metrics.ResettingTimerSnapshot) {
	metricType := dto.MetricType_SUMMARY
	family := rg.getOrCreateFamily(familyMap, name, metricType)

	pv := []float64{0.5, 0.75, 0.95, 0.99, 0.999, 0.9999}
	ps := t.Percentiles(pv)

	quantiles := make([]*dto.Quantile, len(pv))
	for i := range pv {
		quantile := pv[i]
		value := ps[i]
		quantiles[i] = &dto.Quantile{
			Quantile: &quantile,
			Value:    &value,
		}
	}

	count := uint64(t.Count())
	metric := &dto.Metric{
		Summary: &dto.Summary{
			SampleCount: &count,
			Quantile:    quantiles,
		},
	}
	family.Metric = append(family.Metric, metric)
}

// getOrCreateFamily gets or creates a MetricFamily in the map
func (rg *RegistryGatherer) getOrCreateFamily(familyMap map[string]*dto.MetricFamily, name string, metricType dto.MetricType) *dto.MetricFamily {
	family, exists := familyMap[name]
	if !exists {
		family = &dto.MetricFamily{
			Name: &name,
			Type: &metricType,
		}
		familyMap[name] = family
	}
	return family
}

// mutateKey replaces characters not allowed in Prometheus metric names
func mutateKey(key string) string {
	return strings.ReplaceAll(key, "/", "_")
}

// applyPrefix adds the configured prefix to a metric name if set
func (rg *RegistryGatherer) applyPrefix(name string) string {
	if rg.prefix != "" {
		return rg.prefix + "_" + name
	}
	return name
}

// StartPrometheusPusher starts a background goroutine that periodically pushes metrics to Prometheus Pushgateway.
// Returns a cleanup function that should be called to stop the pusher gracefully.
func StartPrometheusPusher(ctx context.Context, addr string, port int, jobName string, prefix string, instance string, updateInterval time.Duration, registry metrics.Registry) func() {
	gatherer := NewRegistryGatherer(registry, prefix)
	pushgatewayURL := fmt.Sprintf("http://%s:%d", addr, port)

	// Create pusher once with optional instance grouping
	pusher := push.New(pushgatewayURL, jobName).Gatherer(gatherer)
	if instance != "" {
		pusher = pusher.Grouping("instance", instance)
	}

	pusherCtx, cancel := context.WithCancel(ctx)
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()

		ticker := time.NewTicker(updateInterval)
		defer ticker.Stop()

		// Push immediately on start
		if err := pusher.Push(); err != nil {
			log.Error("Failed to push metrics to Prometheus Pushgateway", "url", pushgatewayURL, "error", err)
		} else {
			log.Info("Successfully pushed metrics to Prometheus Pushgateway", "url", pushgatewayURL, "job", jobName)
		}

		for {
			select {
			case <-pusherCtx.Done():
				log.Info("Stopping Prometheus Pushgateway pusher")
				return
			case <-ticker.C:
				if err := pusher.Push(); err != nil {
					log.Error("Failed to push metrics to Prometheus Pushgateway", "url", pushgatewayURL, "error", err)
				}
			}
		}
	}()

	return func() {
		cancel()
		wg.Wait()
	}
}
