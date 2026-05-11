// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package transactionfeed

import (
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
	ClientTimeout    time.Duration `koanf:"client-timeout"`
	HandshakeTimeout time.Duration `koanf:"handshake-timeout"`
}

var DefaultServerConfig = ServerConfig{
	Enable:           false,
	Addr:             "",
	Port:             "9646",
	ClientBuf:        256,
	BroadcastBuf:     4096,
	WriteTimeout:     2 * time.Second,
	PingInterval:     30 * time.Second,
	ClientTimeout:    60 * time.Second,
	HandshakeTimeout: 5 * time.Second,
}

func ServerConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".enable", DefaultServerConfig.Enable, "enable transaction feed server")
	f.String(prefix+".addr", DefaultServerConfig.Addr, "address to bind the transaction feed server")
	f.String(prefix+".port", DefaultServerConfig.Port, "port for transaction feed server")
	f.Int(prefix+".client-buf", DefaultServerConfig.ClientBuf, "per-client send buffer size")
	f.Int(prefix+".broadcast-buf", DefaultServerConfig.BroadcastBuf, "broadcast channel buffer size")
	f.Duration(prefix+".write-timeout", DefaultServerConfig.WriteTimeout, "write timeout per client")
	f.Duration(prefix+".ping-interval", DefaultServerConfig.PingInterval, "websocket ping interval")
	f.Duration(prefix+".client-timeout", DefaultServerConfig.ClientTimeout, "client read timeout")
	f.Duration(prefix+".handshake-timeout", DefaultServerConfig.HandshakeTimeout, "websocket handshake timeout")
}
