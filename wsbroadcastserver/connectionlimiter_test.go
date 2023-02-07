// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package wsbroadcastserver

import (
	"net"
	"runtime"
	"testing"

	"github.com/offchainlabs/nitro/util/testhelpers"
)

func TestIpv4BasicConnectionLimiting(t *testing.T) {
	configFetcher := func() *ConnectionLimiterConfig {
		return &ConnectionLimiterConfig{
			Enable:             true,
			PerIpLimit:         3,
			PerIpv6Cidr48Limit: 1,
			PerIpv6Cidr64Limit: 1,
		}
	}
	l := NewConnectionLimiter(configFetcher)

	ip1 := net.ParseIP("1.2.3.4")

	Expect(t, l.IsAllowed(ip1))
	Expect(t, l.Register(ip1))
	Expect(t, l.Register(ip1))
	Expect(t, l.Register(ip1))
	Expect(t, !l.Register(ip1))
	Expect(t, !l.IsAllowed(ip1))

	l.Release(ip1)
	Expect(t, l.IsAllowed(ip1))
	Expect(t, l.Register(ip1))
	Expect(t, !l.IsAllowed(ip1))
	Expect(t, !l.Register(ip1))

	l.Release(ip1)
	l.Release(ip1)
	Expect(t, l.IsAllowed(ip1))
	Expect(t, l.Register(ip1))
	Expect(t, l.IsAllowed(ip1))
	Expect(t, l.Register(ip1))
	Expect(t, !l.IsAllowed(ip1))
	Expect(t, !l.Register(ip1))

	l.Release(ip1)
	l.Release(ip1)
	l.Release(ip1)
	Expect(t, l.IsAllowed(ip1))
	Expect(t, l.Register(ip1))
	Expect(t, l.IsAllowed(ip1))
	Expect(t, l.Register(ip1))
	Expect(t, l.IsAllowed(ip1))
	Expect(t, l.Register(ip1))
	Expect(t, !l.IsAllowed(ip1))
	Expect(t, !l.Register(ip1))
}

func TestTooManyReleases(t *testing.T) {
	configFetcher := func() *ConnectionLimiterConfig {
		return &ConnectionLimiterConfig{
			Enable:             true,
			PerIpLimit:         3,
			PerIpv6Cidr48Limit: 1,
			PerIpv6Cidr64Limit: 1,
		}
	}
	l := NewConnectionLimiter(configFetcher)

	ip1 := net.ParseIP("1.2.3.4")

	Expect(t, l.IsAllowed(ip1))
	Expect(t, l.Register(ip1))
	Expect(t, l.Register(ip1))
	Expect(t, l.Register(ip1))
	Expect(t, !l.Register(ip1))
	Expect(t, !l.IsAllowed(ip1))

	// Make sure the count doesn't go negative and allow too many connections.
	l.Release(ip1)
	l.Release(ip1)
	l.Release(ip1)
	l.Release(ip1)
	l.Release(ip1)
	l.Release(ip1)

	Expect(t, l.IsAllowed(ip1))
	Expect(t, l.Register(ip1))
	Expect(t, l.IsAllowed(ip1))
	Expect(t, l.Register(ip1))
	Expect(t, l.IsAllowed(ip1))
	Expect(t, l.Register(ip1))
	Expect(t, !l.IsAllowed(ip1))
	Expect(t, !l.Register(ip1))
}

func TestIpv6Masks(t *testing.T) {
	configFetcher := func() *ConnectionLimiterConfig {
		return &ConnectionLimiterConfig{
			Enable:             true,
			PerIpLimit:         3,
			PerIpv6Cidr48Limit: 5,
			PerIpv6Cidr64Limit: 2,
		}
	}
	l := NewConnectionLimiter(configFetcher)

	ip1 := net.ParseIP("1:2:3:4:5:5:5:5")
	ip2 := net.ParseIP("1:2:3:4:6:6:6:6")
	ip3 := net.ParseIP("1:2:3:4:7:7:7:7")

	ip4 := net.ParseIP("1:2:3:5:5:5:5:5")
	ip5 := net.ParseIP("1:2:3:6:6:6:6:6")
	ip6 := net.ParseIP("1:2:3:7:7:7:7:7")

	ip7 := net.ParseIP("1:2:4:5:5:5:5:5")
	ip8 := net.ParseIP("1:2:4:6:6:6:6:6")
	ip9 := net.ParseIP("1:2:4:7:7:7:7:7")

	// /64 limit blocks
	Expect(t, l.Register(ip1))  // 1:2:3:4/64 1, 1:2:3/48 1
	Expect(t, l.Register(ip2))  // 1:2:3:4/64 2, 1:2:3/48 2
	Expect(t, !l.Register(ip2)) // 1:2:3:4/64 2*, 1:2:3/48 2
	Expect(t, !l.Register(ip3)) // 1:2:3:4/64 2*, 1:2:3/48 2

	// /48 limit blocks
	Expect(t, l.Register(ip4))  // 1:2:3:5/64 1, 1:2:3/48 3
	Expect(t, l.Register(ip5))  // 1:2:3:6/64 1, 1:2:3/48 4
	Expect(t, l.Register(ip4))  // 1:2:3:5/64 2, 1:2:3/48 5
	Expect(t, !l.Register(ip5)) // 1:2:3:6/64 1, 1:2:3/48 5*

	// /64 limit blocks after releasing from the /48 that would've blocked
	l.Release(ip1)              // 1:2:3:4/64 1, 1:2:3/48 4
	Expect(t, l.Register(ip5))  // 1:2:3:6/64 2, 1:2:3/48 5
	l.Release(ip2)              // 1:2:3:4/64 0, 1:2:3/48 4
	Expect(t, !l.Register(ip5)) // 1:2:3:6/64 2*, 1:2:3/48 4

	// /48 limit blocks a new /64 IP
	Expect(t, l.Register(ip6))  // 1:2:3:7/64 1, 1:2:3/48 5
	Expect(t, !l.Register(ip6)) // 1:2:3:7/64 1, 1:2:3/48 5*

	// IPs in different range to above have separate counts
	Expect(t, l.Register(ip7))  // 1:2:4:5/64 1, 1:2:4/48 1
	Expect(t, l.Register(ip7))  // 1:2:4:5/64 2, 1:2:4/48 2
	Expect(t, !l.Register(ip7)) // 1:2:4:5/64 2*, 1:2:4/48 2
	Expect(t, l.Register(ip8))  // 1:2:4:6/64 1, 1:2:4/48 3
	Expect(t, l.Register(ip8))  // 1:2:4:6/64 2, 1:2:4/48 4
	Expect(t, !l.Register(ip8)) // 1:2:4:6/64 2*, 1:2:4/48 4
	Expect(t, l.Register(ip9))  // 1:2:4:7/64 1, 1:2:4/48 5
	Expect(t, !l.Register(ip9)) // 1:2:4:7/64 1, 1:2:4/48 5

}

func TestPrivateAddresses(t *testing.T) {
	configFetcher := func() *ConnectionLimiterConfig {
		return &ConnectionLimiterConfig{
			Enable:             true,
			PerIpLimit:         1,
			PerIpv6Cidr48Limit: 1,
			PerIpv6Cidr64Limit: 1,
		}
	}
	l := NewConnectionLimiter(configFetcher)
	ip1 := net.ParseIP("fc00:0:0:0:0:0:0:1")
	ip2 := net.ParseIP("fc00:0:0:0:1:0:0:1")

	ip3 := net.ParseIP("10.0.0.1")
	Expect(t, l.Register(ip1))
	Expect(t, l.Register(ip1))
	Expect(t, l.Register(ip1))

	Expect(t, l.Register(ip2))
	Expect(t, l.Register(ip2))
	Expect(t, l.Register(ip2))

	Expect(t, l.Register(ip3))
	Expect(t, l.Register(ip3))
	Expect(t, l.Register(ip3))
}

func Expect(t *testing.T, res bool, text ...interface{}) {
	t.Helper()
	if !res {
		buf := make([]byte, 1<<16)
		stackSize := runtime.Stack(buf, false)
		testhelpers.FailImpl(t, string(buf[0:stackSize]))
	}
}

func Require(t *testing.T, err error, text ...interface{}) {
	t.Helper()
	testhelpers.RequireImpl(t, err, text...)
}

func Fail(t *testing.T, printables ...interface{}) {
	t.Helper()
	testhelpers.FailImpl(t, printables...)
}
