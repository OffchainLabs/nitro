package a

type Config struct {
	L2       int `koanf:"chain"`
	LogLevel int `koanf:"log-level"`
	LogType  int `koanf:"log-type"`
	Metrics  int `koanf:"metrics"`
	PProf    int `koanf:"pprof"`
	Node     int `koanf:"node"`
	Queue    int `koanf:"queue"`
}
