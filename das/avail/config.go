package avail

import (
	"time"
)

type DAConfig struct {
	Enable        bool          `koanf:"enable"`
	AvailApiURL   string        `koanf:"avail-api-url"`
	Seed          string        `koanf:"seed"`
	AppID         int           `koanf:"app-id"`
	Timeout       time.Duration `koanf:"timeout"`
	VectorX       string        `koanf:"vectorx"`
	ArbSepoliaRPC string        `koanf:"arbsepolia-rpc"`
}

func NewDAConfig(avail_api_url string, seed string, app_id int, timeout time.Duration, vectorx string, arbSepolia_rpc string) (*DAConfig, error) {
	return &DAConfig{
		Enable:        true,
		AvailApiURL:   avail_api_url,
		Seed:          seed,
		AppID:         app_id,
		Timeout:       timeout,
		VectorX:       vectorx,
		ArbSepoliaRPC: arbSepolia_rpc,
	}, nil
}
