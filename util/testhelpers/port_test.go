package testhelpers

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
