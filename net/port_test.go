package net

import (
	"testing"
)

func TestFreeTCPPort(t *testing.T) {
	aPort, err := FreeTCPPort()
	if err != nil {
		t.Fatal(err)
	}
	bPort, err := FreeTCPPort()
	if err != nil {
		t.Fatal(err)
	}
	if aPort == bPort {
		t.Errorf("FreeTCPPort() got same port: %v, %v", aPort, bPort)
	}
	if aPort == 0 || bPort == 0 {
		t.Errorf("FreeTCPPort() got port 0")
	}
}

func TestConcurrentFreeTCPPort(t *testing.T) {
	ports := make(chan int, 100)
	errs := make(chan error, 100)
	for i := 0; i < 100; i++ {
		go func() {
			port, err := FreeTCPPort()
			if err != nil {
				errs <- err
				return
			}
			ports <- port
		}()
	}
	seen := make(map[int]bool)
	for i := 0; i < 100; i++ {
		select {
		case port := <-ports:
			if port == 0 {
				t.Errorf("FreeTCPPort() got port 0")
			}
			if seen[port] {
				t.Errorf("FreeTCPPort() got duplicate port: %v", port)
			}
			seen[port] = true
		case err := <-errs:
			t.Fatal(err)
		}
	}
}
