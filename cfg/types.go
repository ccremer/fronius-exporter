package cfg

import "time"

type (
	// Configuration holds a strongly-typed tree of the configuration
	Configuration struct {
		Log      LogConfig
		Symo     SymoConfig
		BindAddr string
	}
	// LogConfig configures the logging options
	LogConfig struct {
		Level   string
		Verbose bool
	}
	// SymoConfig configures the Fronius Symo device
	SymoConfig struct {
		URL     string
		Timeout time.Duration
		Headers []string `mapstructure:"header"`
	}
)

// NewDefaultConfig retrieves the hardcoded configs with sane defaults
func NewDefaultConfig() *Configuration {
	return &Configuration{
		Log: LogConfig{
			Level: "info",
		},
		Symo: SymoConfig{
			URL:     "http://symo.ip.or.hostname/solar_api/v1/GetPowerFlowRealtimeData.fcgi",
			Timeout: 5 * time.Second,
			Headers: []string{},
		},
		BindAddr: ":8080",
	}
}
