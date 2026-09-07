// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package mevfeed

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/pflag"
)

// Config controls the optional local canonical block feed. The feed is deliberately
// disabled by default so an unconfigured node has the same execution behaviour as
// an upstream Nitro node.
type Config struct {
	Enable        bool          `koanf:"enable"`
	Required      bool          `koanf:"required"`
	SocketPath    string        `koanf:"socket-path"`
	SocketMode    uint32        `koanf:"socket-mode"`
	QueueSize     int           `koanf:"queue-size"`
	MaxFrameBytes uint32        `koanf:"max-frame-bytes"`
	WriteTimeout  time.Duration `koanf:"write-timeout"`
	ChainID       uint64        `koanf:"chain-id"`
}

var DefaultConfig = Config{
	SocketPath:    "/run/robinhood-mev/feed.sock",
	SocketMode:    0660,
	QueueSize:     128,
	MaxFrameBytes: 32 * 1024 * 1024,
	WriteTimeout:  50 * time.Millisecond,
}

func ConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".enable", DefaultConfig.Enable, "enable the optional Robinhood MEV canonical block feed")
	f.Bool(prefix+".required", DefaultConfig.Required, "fail node startup when the MEV feed socket cannot be created")
	f.String(prefix+".socket-path", DefaultConfig.SocketPath, "Unix socket path for the MEV feed")
	f.Uint32(prefix+".socket-mode", DefaultConfig.SocketMode, "Unix socket permission mode")
	f.Int(prefix+".queue-size", DefaultConfig.QueueSize, "bounded MEV feed ingress queue size")
	f.Uint32(prefix+".max-frame-bytes", DefaultConfig.MaxFrameBytes, "maximum encoded MEV feed frame size")
	f.Duration(prefix+".write-timeout", DefaultConfig.WriteTimeout, "per-frame Unix socket write timeout")
	f.Uint64(prefix+".chain-id", DefaultConfig.ChainID, "chain ID encoded in feed frames (0 uses the chain config)")
}

func (c Config) Validate() error {
	if !c.Enable {
		return nil
	}
	if !filepath.IsAbs(c.SocketPath) {
		return fmt.Errorf("mev-feed socket-path must be absolute: %q", c.SocketPath)
	}
	if c.SocketMode == 0 || c.SocketMode&^0777 != 0 {
		return fmt.Errorf("mev-feed socket-mode must be a Unix permission mode: %o", c.SocketMode)
	}
	if c.QueueSize < 16 || c.QueueSize > 4096 {
		return fmt.Errorf("mev-feed queue-size must be between 16 and 4096: %d", c.QueueSize)
	}
	if c.MaxFrameBytes < 1024 || c.MaxFrameBytes > 256*1024*1024 {
		return fmt.Errorf("mev-feed max-frame-bytes must be between 1024 and 268435456: %d", c.MaxFrameBytes)
	}
	if c.WriteTimeout <= 0 {
		return fmt.Errorf("mev-feed write-timeout must be positive")
	}
	if c.SocketMode&0002 != 0 {
		return fmt.Errorf("mev-feed socket-mode must not be world writable")
	}
	info, err := os.Stat(filepath.Dir(c.SocketPath))
	if err != nil {
		return fmt.Errorf("mev-feed socket parent: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("mev-feed socket parent is not a directory")
	}
	if info.Mode().Perm()&0002 != 0 {
		return fmt.Errorf("mev-feed socket parent is world writable")
	}
	return nil
}
