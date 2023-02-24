// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package containers

import (
	"sync/atomic"

	"github.com/ethereum/go-ethereum/metrics"
)

type SwappableSample struct {
	sample atomic.Pointer[metrics.Sample]
}

// Assert that *SwappableSample implements metrics.Sample
var _ metrics.Sample = (*SwappableSample)(nil)

func NewSwappableSample() *SwappableSample {
	sample := &SwappableSample{}
	sample.SetSample(&metrics.NilSample{})
	return sample
}

func (s *SwappableSample) GetSample() metrics.Sample {
	return *s.sample.Load()
}

func (s *SwappableSample) SetSample(sample metrics.Sample) {
	s.sample.Store(&sample)
}

func (s *SwappableSample) Clear() {
	s.GetSample().Clear()
}

func (s *SwappableSample) Count() int64 {
	return s.GetSample().Count()
}

func (s *SwappableSample) Max() int64 {
	return s.GetSample().Max()
}

func (s *SwappableSample) Mean() float64 {
	return s.GetSample().Mean()
}

func (s *SwappableSample) Min() int64 {
	return s.GetSample().Min()
}

func (s *SwappableSample) Percentile(p float64) float64 {
	return s.GetSample().Percentile(p)
}

func (s *SwappableSample) Percentiles(p []float64) []float64 {
	return s.GetSample().Percentiles(p)
}

func (s *SwappableSample) Size() int {
	return s.GetSample().Size()
}

func (s *SwappableSample) Snapshot() metrics.Sample {
	return s.GetSample().Snapshot()
}

func (s *SwappableSample) StdDev() float64 {
	return s.GetSample().StdDev()
}

func (s *SwappableSample) Sum() int64 {
	return s.GetSample().Sum()
}

func (s *SwappableSample) Update(x int64) {
	s.GetSample().Update(x)
}

func (s *SwappableSample) Values() []int64 {
	return s.GetSample().Values()
}

func (s *SwappableSample) Variance() float64 {
	return s.GetSample().Variance()
}
