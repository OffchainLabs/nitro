package avail

type DAConfig struct {
	Enable bool   `koanf:"enable"`
	ApiURL string `koanf:"api_url"`
	Seed   string `koanf:"seed"`
	AppID  int    `koanf:"app_id"`
}

func NewDAConfig(api_url string, seed string, app_id int) (*DAConfig, error) {
	return &DAConfig{
		Enable: true,
		ApiURL: api_url,
		Seed:   seed,
		AppID:  app_id,
	}, nil
}
