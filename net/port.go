// Package net provides facilities for working with network connections.
package net

import (
	"net"
)

// FreeTCPPort returns an unused local port.
//
// This is useful for tests that need to bind to a port without risking a conflict.
//
// While, in general, it is not possible to guarantee that the port will remain free
// after the funciton returns, operating systems generally try not to reuse recently
// bound ports until it runs out of free ones. So, this function will, in practice,
// not race with other calls to it.
//
// There is still a possibility that the port will be taken by another process which
// is hardcoded to use a specific port, but that should be extremely rare in tests
// running either locally or in a CI environment.
//
// By separating the port selection out from the code that brings up a server,
// code which uses this function will be more modular and have cleaner separation
// of concerns.
func FreeTCPPort() (int, error) {
	// This works because the kernel will assign an unused port when ":0" is opened.
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}
