package nitroversionalerter

import (
	"context"

	"github.com/spf13/pflag"
)

type ServerConfig struct {
	Enable                    bool   `koanf:"enable"`
	MinRequiredNitroByVersion string `koanf:"min-required-nitro-by-version" reload:"hot"`
	MinRequiredNitroByDate    string `koanf:"min-required-nitro-by-date" reload:"hot"`
	UpgradeDeadline           string `koanf:"upgrade-deadline" reload:"hot"`
}

type ServerConfigFetcher func() *ServerConfig

var DefaultServerConfig = ServerConfig{
	Enable:                    false,
	MinRequiredNitroByVersion: "",
	MinRequiredNitroByDate:    "",
	UpgradeDeadline:           "",
}

func ServerConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".enable", DefaultServerConfig.Enable, "enable arb_getMinRequiredNitroVersion endpoint that returns minimum required version of the nitro node software")
	f.String(prefix+".min-required-nitro-by-version", DefaultServerConfig.MinRequiredNitroByVersion, "minimum required version of the nitro node software. First string in the result of querying arb_getMinRequiredNitroVersion endpoint")
	f.String(prefix+".min-required-nitro-by-date", DefaultServerConfig.MinRequiredNitroByDate, "minimum required version of the nitro node software by date. Second string in the result of querying arb_getMinRequiredNitroVersion endpoint")
	f.String(prefix+".upgrade-deadline", DefaultServerConfig.UpgradeDeadline, "deadline to upgrade the nitro node software. Third string in the result of querying arb_getMinRequiredNitroVersion endpoint")
}

type Server struct {
	config ServerConfigFetcher
}

func NewServer(cfg ServerConfigFetcher) *Server {
	return &Server{cfg}
}

type MinRequiredNitroVersionResult struct {
	NodeVersion     string `json:"nodeVersion"`
	NodeVersionDate string `json:"nodeVersionDate"`
	UpgradeDeadline string `json:"upgradeDeadline"`
}

func (s *Server) GetMinRequiredNitroVersion(ctx context.Context) MinRequiredNitroVersionResult {
	cfg := s.config()
	return MinRequiredNitroVersionResult{cfg.MinRequiredNitroByVersion, cfg.MinRequiredNitroByDate, cfg.UpgradeDeadline}
}
