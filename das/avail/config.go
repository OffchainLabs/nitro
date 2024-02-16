package avail

import "time"

type DAConfig struct {
	Enable  bool          `koanf:"enable"`
	ApiURL  string        `koanf:"api_url"`
	Seed    string        `koanf:"seed"`
	AppID   int           `koanf:"app_id"`
	Timeout time.Duration `koanf:"timeout"`
}

func NewDAConfig(api_url string, seed string, app_id int, timeout time.Duration) (*DAConfig, error) {
	return &DAConfig{
		Enable:  true,
		ApiURL:  api_url,
		Seed:    seed,
		AppID:   app_id,
		Timeout: timeout,
	}, nil
}
