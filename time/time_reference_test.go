// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE
package time

import (
	"testing"
	"time"
)

var (
	_ = Reference(&realTimeReference{})
	_ = Reference(&ArtificialTimeReference{})
)

func TestRealTimeReference(t *testing.T) {
	rtRef := NewRealTimeReference()
	now := rtRef.Get()
	time.Sleep(time.Millisecond)
	newTime := rtRef.Get()
	if newTime.Before(now) || newTime.Equal(now) {
		t.Errorf("Time did not advance as expected")
	}
}

func TestRealTimeReference_SleepUntil(t *testing.T) {
	rtRef := NewRealTimeReference()
	wakeTime := rtRef.Get().Add(time.Millisecond * 10)
	rtRef.SleepUntil(wakeTime)
	now := rtRef.Get()
	if now.Before(wakeTime) {
		t.Errorf("SleepUntil did not sleep until the correct time")
	}
}

func TestRealTimeReference_NewTicker(t *testing.T) {
	rtRef := NewRealTimeReference()
	ticker := rtRef.NewTicker(time.Millisecond * 10)
	time.Sleep(time.Millisecond * 15)
	select {
	case <-ticker.C():
	// successfully ticked
	default:
		t.Errorf("Ticker did not tick as expected")
	}
	ticker.Stop()
}

func TestArtificialTimeReference_GetSet(t *testing.T) {
	atRef := NewArtificialTimeReference()
	time1 := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	atRef.Set(time1)
	time2 := atRef.Get()
	if !time1.Equal(time2) {
		t.Errorf("Did not get/set time as expected")
	}
}

func TestArtificialTimeReference_SleepUntil(t *testing.T) {
	atRef := NewArtificialTimeReference()
	sleepUntil := time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)
	go func() {
		time.Sleep(time.Millisecond * 10)
		atRef.Set(sleepUntil)
	}()
	atRef.SleepUntil(sleepUntil)
	if atRef.Get().Before(sleepUntil) {
		t.Errorf("SleepUntil did not sleep as expected")
	}
}

func TestRealTimeReference_Sleep(t *testing.T) {
	rtRef := NewRealTimeReference()
	start := rtRef.Get()
	sleepDuration := time.Millisecond * 10
	rtRef.Sleep(sleepDuration)
	end := rtRef.Get()
	if end.Sub(start) < sleepDuration {
		t.Errorf("Sleep did not last for the expected duration")
	}
}

func TestArtificialTimeReference_Sleep(t *testing.T) {
	atRef := NewArtificialTimeReference()
	start := atRef.Get()
	sleepDuration := time.Minute
	go func() {
		time.Sleep(time.Millisecond * 10)
		atRef.Add(sleepDuration)
	}()
	atRef.Sleep(sleepDuration)
	end := atRef.Get()
	if end.Sub(start) < sleepDuration {
		t.Errorf("Sleep did not last for the expected duration")
	}
}

func TestArtificialTicker(t *testing.T) {
	atRef := NewArtificialTimeReference()
	interval := time.Minute
	ticker := atRef.NewTicker(interval)
	go func() {
		time.Sleep(time.Millisecond * 10)
		atRef.Add(interval)
	}()
	select {
	case <-ticker.C():
	// Ticker ticked
	case <-time.After(time.Second):
		t.Errorf("Ticker did not tick as expected")
	}
	ticker.Stop()
}
