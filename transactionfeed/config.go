// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package transactionfeed

import (
	"fmt"
	"time"

	"github.com/spf13/pflag"
)

type ServerConfig struct {
	Enable           bool          `koanf:"enable"`
	Addr             string        `koanf:"addr"`
	Port             string        `koanf:"port"`
	ClientBuf        int           `koanf:"client-buf"`
	BroadcastBuf     int           `koanf:"broadcast-buf"`
	WriteTimeout     time.Duration `koanf:"write-timeout"`
	PingInterval     time.Duration `koanf:"ping-interval"`
	ReadTimeout      time.Duration `koanf:"read-timeout"`
	HandshakeTimeout time.Duration `koanf:"handshake-timeout"`
}

var DefaultServerConfig = ServerConfig{
	Enable:           false,
	Addr:             "",
	Port:             "9646",
	ClientBuf:        256,
	BroadcastBuf:     4096,
	ReadTimeout:      2 * time.Second,
	WriteTimeout:     2 * time.Second,
	PingInterval:     30 * time.Second,
	HandshakeTimeout: 5 * time.Second,
}

func ServerConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".enable", DefaultServerConfig.Enable, "enable transaction feed server")
	f.String(prefix+".addr", DefaultServerConfig.Addr, "address to bind the transaction feed server")
	f.String(prefix+".port", DefaultServerConfig.Port, "port for transaction feed server")
	f.Int(prefix+".client-buf", DefaultServerConfig.ClientBuf, "per-client send buffer size")
	f.Int(prefix+".broadcast-buf", DefaultServerConfig.BroadcastBuf, "broadcast channel buffer size")
	f.Duration(prefix+".read-timeout", DefaultServerConfig.ReadTimeout, "read timeout per client")
	f.Duration(prefix+".write-timeout", DefaultServerConfig.WriteTimeout, "write timeout per client")
	f.Duration(prefix+".ping-interval", DefaultServerConfig.PingInterval, "websocket ping interval")
	f.Duration(prefix+".handshake-timeout", DefaultServerConfig.HandshakeTimeout, "websocket handshake timeout")
}

func (c *ServerConfig) Validate() error {
	if !c.Enable {
		return nil
	}
	if c.Port == "" {
		return fmt.Errorf("transactionfeed: port must be set when enabled")
	}
	if c.ClientBuf <= 0 {
		return fmt.Errorf("transactionfeed: client-buf must be > 0 (got %d)", c.ClientBuf)
	}
	if c.BroadcastBuf <= 0 {
		return fmt.Errorf("transactionfeed: broadcast-buf must be > 0 (got %d)", c.BroadcastBuf)
	}
	if c.ReadTimeout <= 0 {
		return fmt.Errorf("transactionfeed: read-timeout must be > 0 (got %s)", c.ReadTimeout)
	}
	if c.WriteTimeout <= 0 {
		return fmt.Errorf("transactionfeed: write-timeout must be > 0 (got %s)", c.WriteTimeout)
	}
	if c.PingInterval <= 0 {
		return fmt.Errorf("transactionfeed: ping-interval must be > 0 (got %s)", c.PingInterval)
	}
	if c.HandshakeTimeout <= 0 {
		return fmt.Errorf("transactionfeed: handshake-timeout must be > 0 (got %s)", c.HandshakeTimeout)
	}
	return nil
}
