// Package net provides facilities for working with network connections.
package net

import (
	"net"
)

// FreeTCPPort returns an unused local port.
//
// This is useful for tests that need to bind to a port without risking a conflict.
func FreeTCPPort() (int, error) {
	// This works because the kernel will assign an unused port when ":0" is opened.
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}
