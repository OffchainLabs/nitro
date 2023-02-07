// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package wsbroadcastserver

import (
	"net"
	"sync"

	"github.com/ethereum/go-ethereum/log"
	flag "github.com/spf13/pflag"
)

type ConnectionLimiterConfig struct {
	Enable             bool `koanf:"enable" reload:"hot"`
	PerIpLimit         int  `koanf:"per-ip-limit" reload:"hot"`
	PerIpv6Cidr48Limit int  `koanf:"per-ipv6-cidr-48-limit" reload:"hot"`
	PerIpv6Cidr64Limit int  `koanf:"per-ipv6-cidr-64-limit" reload:"hot"`
}

var DefaultConnectionLimiterConfig = ConnectionLimiterConfig{
	Enable:             false,
	PerIpLimit:         5,
	PerIpv6Cidr48Limit: 20,
	PerIpv6Cidr64Limit: 10,
}

func ConnectionLimiterConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enable", DefaultConnectionLimiterConfig.Enable, "enable broadcaster per-client connection limiting")
	f.Int(prefix+".per-ip-limit", DefaultConnectionLimiterConfig.PerIpLimit, "limit clients, as identified by IPv4/v6 address, to this many connections to this relay")
	f.Int(prefix+".per-ipv6-cidr-48-limit", DefaultConnectionLimiterConfig.PerIpv6Cidr48Limit, "limit ipv6 clients, as identified by IPv6 address masked with /48, to this many connections to this relay")
	f.Int(prefix+".per-ipv6-cidr-64-limit", DefaultConnectionLimiterConfig.PerIpv6Cidr64Limit, "limit ipv6 clients, as identified by IPv6 address masked with /64, to this many connections to this relay")
}

type ConnectionLimiterConfigFetcher func() *ConnectionLimiterConfig

type ConnectionLimiter struct {
	sync.RWMutex

	ipConnectionCounts map[string]int
	config             ConnectionLimiterConfigFetcher
}

func NewConnectionLimiter(configFetcher ConnectionLimiterConfigFetcher) *ConnectionLimiter {
	return &ConnectionLimiter{
		ipConnectionCounts: make(map[string]int),
		config:             configFetcher,
	}
}

func (l *ConnectionLimiter) IsAllowed(ip net.IP) bool {
	l.RLock()
	defer l.RUnlock()
	return l.isAllowedImpl(ip)
}

func isIpv6(ip net.IP) bool {
	// This seems to be the canonical way to distinguish IPv4 from IPv6 in Go
	// https://stackoverflow.com/questions/22751035/golang-distinguish-ipv4-ipv6
	// We don't care about the case where it is an IPv4 address in IPv6
	// representation, we'll just treat that as IPv4.
	return ip.To4() == nil
}

func (l *ConnectionLimiter) isAllowedImpl(ip net.IP) bool {
	if ip == nil || ip.IsPrivate() || ip.IsLoopback() {
		log.Warn("Ignoring private, looback, or unparseable IP. Please check relay and network configuration to ensure client IP addresses are detected correctly", "ip", ip)
		return true
	}

	config := l.config()

	if res := l.ipConnectionCounts[string(ip)]; res >= config.PerIpLimit {
		return false
	}

	if isIpv6(ip) {
		ipv6Slash48 := ip.Mask(net.CIDRMask(48, 128))
		if ipv6Slash48 == nil {
			log.Warn("Error taking /48 mask of ipv6 client address", "ip", ip)
		} else if res := l.ipConnectionCounts[string(ipv6Slash48)+"/48"]; res >= config.PerIpv6Cidr48Limit {
			return false
		}

		ipv6Slash64 := ip.Mask(net.CIDRMask(64, 128))
		if ipv6Slash64 == nil {
			log.Warn("Error taking /64 mask of ipv6 client address", "ip", ip)
		} else if res := l.ipConnectionCounts[string(ipv6Slash64)+"/64"]; res >= config.PerIpv6Cidr64Limit {
			return false
		}
	}

	return true
}

func (l *ConnectionLimiter) updateUsage(ip net.IP, increment bool) {
	if ip == nil {
		return
	}

	updateAmount := -1
	if increment {
		updateAmount = 1
	}

	config := l.config()
	updateAndCheckBounds := func(ipString string, bound int) {
		l.ipConnectionCounts[ipString] += updateAmount
		if l.ipConnectionCounts[ipString] < 0 {
			log.Error("BUG: Unbalanced ConnectionLimiter.updateUsage(..., false) calls")
			l.ipConnectionCounts[ipString] = 0
		} else if l.ipConnectionCounts[ipString] > bound {
			log.Error("BUG: Unbalanced ConnectionLimiter.updateUsage(..., true) calls")
			l.ipConnectionCounts[ipString] = bound
		}
	}

	updateAndCheckBounds(string(ip), config.PerIpLimit)

	if isIpv6(ip) {
		ipv6Slash48 := ip.Mask(net.CIDRMask(48, 128))
		if ipv6Slash48 != nil {
			updateAndCheckBounds(string(ipv6Slash48)+"/48", config.PerIpv6Cidr48Limit)
		}

		ipv6Slash64 := ip.Mask(net.CIDRMask(64, 128))
		if ipv6Slash64 != nil {
			updateAndCheckBounds(string(ipv6Slash64)+"/64", config.PerIpv6Cidr64Limit)
		}
	}
}

func (l *ConnectionLimiter) Register(ip net.IP) bool {
	l.Lock()
	defer l.Unlock()

	// First check if allowed without modifying counts so that we don't need to roll back partial counts.
	if !l.isAllowedImpl(ip) {
		return false
	}

	l.updateUsage(ip, true)

	return true
}

func (l *ConnectionLimiter) Release(ip net.IP) {
	l.Lock()
	defer l.Unlock()

	l.updateUsage(ip, false)

}
