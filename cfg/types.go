package cfg

import "time"

type (
	// Configuration holds a strongly-typed tree of the configuration
	Configuration struct {
		Log      LogConfig  `koanf:"log"`
		Symo     SymoConfig `koanf:"symo"`
		BindAddr string     `koanf:"bind-addr"`
	}
	// LogConfig configures the logging options
	LogConfig struct {
		Level   string `koanf:"level"`
		Verbose bool   `koanf:"verbose"`
	}
	// SymoConfig configures the Fronius Symo device
	SymoConfig struct {
		URL              string        `koanf:"url"`
		Timeout          time.Duration `koanf:"timeout"`
		Headers          []string      `koanf:"header"`
		PowerFlowEnabled bool          `koanf:"enable-power-flow"`
		ArchiveEnabled   bool          `koanf:"enable-archive"`
	}
)

// NewDefaultConfig retrieves the hardcoded configs with sane defaults
func NewDefaultConfig() *Configuration {
	return &Configuration{
		Log: LogConfig{
			Level: "info",
		},
		Symo: SymoConfig{
			URL:              "http://symo.ip.or.hostname/solar_api/v1/GetPowerFlowRealtimeData.fcgi",
			Timeout:          5 * time.Second,
			Headers:          []string{},
			PowerFlowEnabled: true,
			ArchiveEnabled:   true,
		},
		BindAddr: ":8080",
	}
}
