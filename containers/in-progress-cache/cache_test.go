package inprogresscache

import (
	"errors"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestCompute(t *testing.T) {
	cache := New[string, int]()
	requestId := "testRequest"

	// Define a computation function
	computeFunc := func() (int, error) {
		time.Sleep(100 * time.Millisecond)
		return 42, nil
	}

	// Call Compute and check the result
	result, err := cache.Compute(requestId, computeFunc)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if result != 42 {
		t.Errorf("Expected result to be 42, got %d", result)
	}

	// Call Compute again with the same requestId and ensure the cached value is returned
	cachedResult, cachedErr := cache.Compute(requestId, computeFunc)
	if cachedErr != nil {
		t.Errorf("Expected no error from cached result, got %v", cachedErr)
	}
	if cachedResult != result {
		t.Errorf("Expected cached result to be %d, got %d", result, cachedResult)
	}
}

// TestConcurrentComputations tests that concurrent calls to Compute with the same request ID
// only result in a single computation.
func TestConcurrentComputations(t *testing.T) {
	cache := New[string, int]()
	requestId := "concurrentTest"
	counter := 0

	computeFunc := func() (int, error) {
		time.Sleep(100 * time.Millisecond)
		counter++
		return counter, nil
	}

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := cache.Compute(requestId, computeFunc); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()

	// Verify that the computation was only performed once
	if counter != 1 {
		t.Errorf("Expected a single computation, got %d", counter)
	}
}

// TestConcurrentComputationsWithError tests that concurrent calls to Compute with the same request ID
// only result in a single computation even if the computation returns an error.
// The error should be returned to all goroutines awaiting the computation's completion.
func TestConcurrentComputationsWithError(t *testing.T) {
	cache := New[string, int]()
	requestId := "concurrentTest"
	counter := 0

	computeFunc := func() (int, error) {
		time.Sleep(100 * time.Millisecond)
		counter++
		return 0, errors.New(strconv.Itoa(counter))
	}

	expectedError := errors.New("1")
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := cache.Compute(requestId, computeFunc); err != nil {
				if !strings.Contains(err.Error(), expectedError.Error()) {
					t.Errorf("Expected a single computation, got %s", err.Error())
				}
			}
		}()
	}
	wg.Wait()

	// Verify that the computation was only performed once
	if counter != 1 {
		t.Errorf("Expected a single computation, got %d", counter)
	}
}
