package cfg

import "time"

type (
	// Configuration holds a strongly-typed tree of the configuration
	Configuration struct {
		Log      LogConfig  `koanf:"log"`
		Symo     SymoConfig `koanf:"symo"`
		MQTT     MQTTConfig `koanf:"mqtt"`
		Poll     PollConfig `koanf:"poll"`
		BindAddr string     `koanf:"bind-addr"`
	}
	// LogConfig configures the logging options
	LogConfig struct {
		Level   string `koanf:"level"`
		Verbose bool   `koanf:"verbose"`
	}
	// SymoConfig configures the Fronius Symo device
	SymoConfig struct {
		URL                     string        `koanf:"url"`
		Timeout                 time.Duration `koanf:"timeout"`
		Headers                 []string      `koanf:"header"`
		PowerFlowEnabled        bool          `koanf:"enable-power-flow"`
		ArchiveEnabled          bool          `koanf:"enable-archive"`
		InverterRealtimeEnabled bool          `koanf:"enable-inverter-realtime"`
		MeterRealtimeEnabled    bool          `koanf:"enable-meter-realtime"`
	}
	// MQTTConfig configures MQTT publishing of updates
	MQTTConfig struct {
		Broker                  string        `koanf:"broker"`
		Username                string        `koanf:"username"`
		Password                string        `koanf:"password"`
		BaseTopic               string        `koanf:"base-topic"`
		ClientID                string        `koanf:"client-id"`
		QueueSize               int           `koanf:"queue-size"`
		ReconnectInterval       time.Duration `koanf:"reconnect-interval"`
		AvailabilityTopic       string        `koanf:"availability-topic"`
		AvailabilityPayloadUp   string        `koanf:"availability-payload-up"`
		AvailabilityPayloadDown string        `koanf:"availability-payload-down"`
		DiscoveryEnabled        bool          `koanf:"discovery-enabled"`
		DiscoveryPrefix         string        `koanf:"discovery-prefix"`
		DiscoveryDeviceName     string        `koanf:"discovery-device-name"`
		DiscoveryDeviceID       string        `koanf:"discovery-device-id"`
		TLSCAFile               string        `koanf:"tls-ca-file"`
		TLSCertFile             string        `koanf:"tls-cert-file"`
		TLSKeyFile              string        `koanf:"tls-key-file"`
		TLSServerName           string        `koanf:"tls-server-name"`
		TLSInsecureSkipVerify   bool          `koanf:"tls-insecure-skip-verify"`
	}
	// PollConfig configures background polling and cache freshness
	PollConfig struct {
		Interval     time.Duration `koanf:"interval"`
		FreshTimeout time.Duration `koanf:"fresh-timeout"`
	}
)

// NewDefaultConfig retrieves the hardcoded configs with sane defaults
func NewDefaultConfig() *Configuration {
	return &Configuration{
		Log: LogConfig{
			Level: "info",
		},
		Symo: SymoConfig{
			URL:                     "http://symo.ip.or.hostname",
			Timeout:                 5,
			Headers:                 []string{},
			PowerFlowEnabled:        true,
			ArchiveEnabled:          true,
			InverterRealtimeEnabled: true,
			MeterRealtimeEnabled:    true,
		},
		MQTT: MQTTConfig{
			Broker:                  "",
			Username:                "",
			Password:                "",
			BaseTopic:               "fronius-exporter",
			ClientID:                "",
			QueueSize:               16,
			ReconnectInterval:       5,
			AvailabilityTopic:       "",
			AvailabilityPayloadUp:   "online",
			AvailabilityPayloadDown: "offline",
			DiscoveryEnabled:        false,
			DiscoveryPrefix:         "homeassistant",
			DiscoveryDeviceName:     "Fronius Exporter",
			DiscoveryDeviceID:       "fronius-exporter",
			TLSCAFile:               "",
			TLSCertFile:             "",
			TLSKeyFile:              "",
			TLSServerName:           "",
			TLSInsecureSkipVerify:   false,
		},
		Poll: PollConfig{
			Interval:     10,
			FreshTimeout: 30,
		},
		BindAddr: ":8080",
	}
}
