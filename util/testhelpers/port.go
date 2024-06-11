package testhelpers

import (
	"net"
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
