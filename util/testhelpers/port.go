package testhelpers

import (
	"net"
	"testing"
)

// FreeTCPPortListener returns a listener listening on an unused local port.
//
// This is useful for tests that need to bind to a port without risking a conflict.
func FreeTCPPortListener() (net.Listener, error) {
	// This works because the kernel will assign an unused port when ":0" is opened.
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	return l, nil
}

// Func AddrTCPPort returns the port of a net.Addr.
func AddrTCPPort(n net.Addr, t *testing.T) int {
	t.Helper()
	tcpAddr, ok := n.(*net.TCPAddr)
	if !ok {
		t.Fatal("Could not get TCP address net.Addr")
	}
	return tcpAddr.Port
}
