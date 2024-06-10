package net

import (
	"net"
	"testing"
)

func TestFreeTCPPortListener(t *testing.T) {
	aListener, err := FreeTCPPortListener()
	if err != nil {
		t.Fatal(err)
	}
	bListener, err := FreeTCPPortListener()
	if err != nil {
		t.Fatal(err)
	}
	if aListener.Addr().(*net.TCPAddr).Port == bListener.Addr().(*net.TCPAddr).Port {
		t.Errorf("FreeTCPPortListener() got same port: %v, %v", aListener, bListener)
	}
	if aListener.Addr().(*net.TCPAddr).Port == 0 || bListener.Addr().(*net.TCPAddr).Port == 0 {
		t.Errorf("FreeTCPPortListener() got port 0")
	}
}

func TestConcurrentFreeTCPPort(t *testing.T) {
	ports := make(chan int, 100)
	errs := make(chan error, 100)
	for i := 0; i < 100; i++ {
		go func() {
			l, err := FreeTCPPortListener()
			if err != nil {
				errs <- err
				return
			}
			ports <- l.Addr().(*net.TCPAddr).Port
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
