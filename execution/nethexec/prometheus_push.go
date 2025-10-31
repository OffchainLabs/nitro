// Copyright 2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package nethexec

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
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

const (
	contentTypeHeader = "Content-Type"
	base64Suffix      = "@base64"
)

var errJobEmpty = errors.New("job name is empty")

// nethPusher manages a push to the Pushgateway with custom URL handling
type nethPusher struct {
	error error

	url      string
	job      string
	grouping map[string]string

	gatherers prometheus.Gatherers

	client HTTPDoer

	expfmt expfmt.Format
}

// HTTPDoer is an interface for the one method of http.Client that is used by nethPusher
type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

// newNethPusher creates a new nethPusher to push to the provided URL with the provided job name.
// The URL should already include the /metrics path if needed.
func newNethPusher(pushURL string, job string) *nethPusher {
	var err error
	if job == "" {
		err = errJobEmpty
	}
	if !strings.Contains(pushURL, "://") {
		pushURL = "http://" + pushURL
	}
	pushURL = strings.TrimSuffix(pushURL, "/")

	return &nethPusher{
		error:     err,
		url:       pushURL,
		job:       job,
		grouping:  map[string]string{},
		gatherers: prometheus.Gatherers{},
		client:    &http.Client{},
		expfmt:    expfmt.FmtProtoDelim,
	}
}

// Gatherer adds a Gatherer to the nethPusher
func (p *nethPusher) Gatherer(g prometheus.Gatherer) *nethPusher {
	p.gatherers = append(p.gatherers, g)
	return p
}

// Grouping adds a label pair to the grouping key of the nethPusher
func (p *nethPusher) Grouping(name string, value string) *nethPusher {
	if p.error == nil {
		if !model.LabelName(name).IsValid() {
			p.error = fmt.Errorf("grouping label has invalid name: %s", name)
			return p
		}
		p.grouping[name] = value
	}
	return p
}

// AddContext works like Add but includes a context
func (p *nethPusher) AddContext(ctx context.Context) error {
	return p.push(ctx, http.MethodPost)
}

// Push uses POST method to add/merge metrics
func (p *nethPusher) Push() error {
	return p.push(context.Background(), http.MethodPost)
}

func (p *nethPusher) push(ctx context.Context, method string) error {
	if p.error != nil {
		return p.error
	}
	mfs, err := p.gatherers.Gather()
	if err != nil {
		return err
	}
	buf := &bytes.Buffer{}
	enc := expfmt.NewEncoder(buf, p.expfmt)
	// Check for pre-existing grouping labels:
	for _, mf := range mfs {
		for _, m := range mf.GetMetric() {
			for _, l := range m.GetLabel() {
				if l.GetName() == "job" {
					return fmt.Errorf("pushed metric %s (%s) already contains a job label", mf.GetName(), m)
				}
				if _, ok := p.grouping[l.GetName()]; ok {
					return fmt.Errorf(
						"pushed metric %s (%s) already contains grouping label %s",
						mf.GetName(), m, l.GetName(),
					)
				}
			}
		}
		if err := enc.Encode(mf); err != nil {
			return fmt.Errorf(
				"failed to encode metric family %s, error is %w",
				mf.GetName(), err)
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, p.fullURL(), buf)
	if err != nil {
		return err
	}
	req.Header.Set(contentTypeHeader, string(p.expfmt))
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	// Depending on version and configuration of the PGW, StatusOK or StatusAccepted may be returned.
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body) // Ignore any further error as this is for an error message only.
		return fmt.Errorf("unexpected status code %d while pushing to %s: %s", resp.StatusCode, p.fullURL(), body)
	}
	return nil
}

// fullURL constructs the full URL by appending job and grouping labels to the base URL.
// Unlike the standard Prometheus pusher, this does NOT add /metrics prefix - it expects
// the base URL to already contain it if needed.
func (p *nethPusher) fullURL() string {
	urlComponents := []string{}
	if encodedJob, isBase64 := p.encodeComponent(p.job); isBase64 {
		urlComponents = append(urlComponents, "job"+base64Suffix, encodedJob)
	} else {
		urlComponents = append(urlComponents, "job", encodedJob)
	}
	for ln, lv := range p.grouping {
		if encodedLV, isBase64 := p.encodeComponent(lv); isBase64 {
			urlComponents = append(urlComponents, ln+base64Suffix, encodedLV)
		} else {
			urlComponents = append(urlComponents, ln, encodedLV)
		}
	}
	return fmt.Sprintf("%s/%s", p.url, strings.Join(urlComponents, "/"))
}

// encodeComponent encodes the provided string with base64.RawURLEncoding in
// case it contains '/' and as "=" in case it is empty. If neither is the case,
// it uses url.QueryEscape instead. It returns true in the former two cases.
func (p *nethPusher) encodeComponent(s string) (string, bool) {
	if s == "" {
		return "=", true
	}
	if strings.Contains(s, "/") {
		return base64.RawURLEncoding.EncodeToString([]byte(s)), true
	}
	return url.QueryEscape(s), false
}

// StartPrometheusPusher starts a background goroutine that periodically pushes metrics to Prometheus Pushgateway.
// Returns a cleanup function that should be called to stop the pusher gracefully.
func StartPrometheusPusher(ctx context.Context, pushgatewayURL string, jobName string, prefix string, instance string, updateInterval time.Duration, registry metrics.Registry) func() {
	gatherer := NewRegistryGatherer(registry, prefix)

	// Create pusher once with optional instance grouping
	pusher := newNethPusher(pushgatewayURL, jobName).Gatherer(gatherer)
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
