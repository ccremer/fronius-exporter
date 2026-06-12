package cfg

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/knadh/koanf/v2"

	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/posflag"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
)

// ParseConfig overrides internal config defaults with up CLI flags, environment variables and ensures basic validation.
func ParseConfig(version, commit, date string, fs *flag.FlagSet, args []string) *Configuration {
	config := NewDefaultConfig()

	setupCliFlags(fmt.Sprintf("version %s, %s, %s", version, commit, date), fs, config)

	loadConfigHierarchy(fs, args, config)

	postLoadProcess(config)

	log.WithField("config", *config).Debug("Parsed config")
	return config
}

func setupCliFlags(version string, fs *flag.FlagSet, config *Configuration) {
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s (%s):\n", os.Args[0], version)
		fs.PrintDefaults()
	}
	fs.String("bind-addr", config.BindAddr, "IP Address to bind to listen for Prometheus scrapes.")
	fs.String("log.level", config.Log.Level, "Logging level.")
	fs.BoolP("log.verbose", "v", config.Log.Verbose, "Shortcut for --log.level=debug.")
	fs.StringSlice("symo.header", config.Symo.Headers,
		"List of \"key: value\" headers to append to the requests going to Fronius Symo. Example: --symo.header \"authorization=Basic <base64>\".")
	fs.StringP("symo.url", "u", config.Symo.URL, "Target base URL of Fronius Symo device.")
	fs.Int64("symo.timeout", int64(config.Symo.Timeout.Seconds()),
		"Timeout in seconds when collecting metrics from Fronius Symo. Should not be larger than the scrape interval.")
	fs.Bool("symo.enable-power-flow", config.Symo.PowerFlowEnabled, "Enable/disable scraping of power flow data")
	fs.Bool("symo.enable-archive", config.Symo.ArchiveEnabled, "Enable/disable scraping of archive data")
	fs.Bool("symo.enable-inverter-realtime", config.Symo.InverterRealtimeEnabled, "Enable/disable scraping of inverter real time data")
	fs.Bool("symo.enable-meter-realtime", config.Symo.MeterRealtimeEnabled, "Enable/disable scraping of meter real time data")
	fs.String("mqtt.broker", config.MQTT.Broker, "MQTT broker URL, e.g. tcp://broker:1883. Leave empty to disable MQTT publishing.")
	fs.String("mqtt.username", config.MQTT.Username, "Username for MQTT authentication.")
	fs.String("mqtt.password", config.MQTT.Password, "Password for MQTT authentication.")
	fs.String("mqtt.base-topic", config.MQTT.BaseTopic, "Base MQTT topic used for publishing updates.")
	fs.String("mqtt.client-id", config.MQTT.ClientID, "MQTT client ID. Leave empty to auto-generate one.")
	fs.Int("mqtt.queue-size", config.MQTT.QueueSize, "Size of the asynchronous MQTT publish queue.")
	fs.Int64("mqtt.reconnect-interval", int64(config.MQTT.ReconnectInterval.Seconds()), "MQTT reconnect retry interval in seconds.")
	fs.String("mqtt.availability-topic", config.MQTT.AvailabilityTopic, "MQTT availability topic for Home Assistant and LWT. Defaults to <mqtt.base-topic>/availability when empty.")
	fs.String("mqtt.availability-payload-up", config.MQTT.AvailabilityPayloadUp, "MQTT payload published when exporter is online.")
	fs.String("mqtt.availability-payload-down", config.MQTT.AvailabilityPayloadDown, "MQTT last-will payload published when exporter is offline.")
	fs.Bool("mqtt.discovery-enabled", config.MQTT.DiscoveryEnabled, "Enable Home Assistant MQTT discovery publishing.")
	fs.String("mqtt.discovery-prefix", config.MQTT.DiscoveryPrefix, "Home Assistant MQTT discovery prefix.")
	fs.String("mqtt.discovery-device-name", config.MQTT.DiscoveryDeviceName, "Home Assistant device name for discovered entities.")
	fs.String("mqtt.discovery-device-id", config.MQTT.DiscoveryDeviceID, "Home Assistant device identifier for discovered entities.")
	fs.String("mqtt.tls-ca-file", config.MQTT.TLSCAFile, "CA certificate file for MQTT TLS.")
	fs.String("mqtt.tls-cert-file", config.MQTT.TLSCertFile, "Client certificate file for MQTT TLS.")
	fs.String("mqtt.tls-key-file", config.MQTT.TLSKeyFile, "Client key file for MQTT TLS.")
	fs.String("mqtt.tls-server-name", config.MQTT.TLSServerName, "Override TLS server name for MQTT.")
	fs.Bool("mqtt.tls-insecure-skip-verify", config.MQTT.TLSInsecureSkipVerify, "Skip MQTT TLS certificate verification.")
	fs.Int64("poll.interval", int64(config.Poll.Interval.Seconds()), "Background polling interval in seconds.")
	fs.Int64("poll.fresh-timeout", int64(config.Poll.FreshTimeout.Seconds()), "Maximum age in seconds for cached data served by HTTP handlers.")
}

func postLoadProcess(config *Configuration) {
	config.Symo.Timeout *= time.Second
	if config.Log.Verbose {
		config.Log.Level = "debug"
	}
	config.MQTT.BaseTopic = strings.Trim(config.MQTT.BaseTopic, "/")
	config.MQTT.DiscoveryPrefix = strings.Trim(config.MQTT.DiscoveryPrefix, "/")
	config.MQTT.DiscoveryDeviceID = strings.TrimSpace(config.MQTT.DiscoveryDeviceID)
	config.MQTT.DiscoveryDeviceName = strings.TrimSpace(config.MQTT.DiscoveryDeviceName)
	config.MQTT.AvailabilityTopic = strings.Trim(config.MQTT.AvailabilityTopic, "/")
	if config.MQTT.AvailabilityTopic == "" && config.MQTT.BaseTopic != "" {
		config.MQTT.AvailabilityTopic = config.MQTT.BaseTopic + "/availability"
	}
	config.MQTT.ReconnectInterval *= time.Second
	if config.MQTT.QueueSize <= 0 {
		config.MQTT.QueueSize = 16
	}
	if config.MQTT.ReconnectInterval <= 0 {
		config.MQTT.ReconnectInterval = 5 * time.Second
	}
	config.Poll.Interval *= time.Second
	config.Poll.FreshTimeout *= time.Second
	if config.Poll.Interval <= 0 {
		config.Poll.Interval = 10 * time.Second
	}
	if config.Poll.FreshTimeout <= 0 {
		config.Poll.FreshTimeout = 30 * time.Second
	}

	var parsedHeaders []string
	for _, header := range config.Symo.Headers {
		parsedHeaders = splitHeaderStrings(header, parsedHeaders)
	}
	config.Symo.Headers = parsedHeaders

	level, err := log.ParseLevel(config.Log.Level)
	if err != nil {
		log.WithError(err).Warn("Could not parse log level, fallback to info level")
		config.Log.Level = "info"
		log.SetLevel(log.InfoLevel)
	} else {
		log.SetLevel(level)
	}
}

func splitHeaderStrings(rest string, headers []string) []string {
	s := strings.TrimPrefix(rest, ",")
	arr := strings.SplitN(s, ",", 2)
	if v := arr[0]; v != "" {
		headers = append(headers, strings.TrimSpace(v))
	}
	if len(arr) < 2 {
		// No more key-value pairs to parse
		return headers
	}
	return splitHeaderStrings(arr[1], headers)
}

func loadConfigHierarchy(fs *flag.FlagSet, args []string, config *Configuration) {
	koanfInstance := koanf.New(".")

	// Environment variables
	if err := koanfInstance.Load(env.Provider("", ".", func(s string) string {
		/*
			Configuration can contain hierarchies (YAML, etc.) and CLI flags dashes.
			To read environment variables with hierarchies and dashes we replace the hierarchy delimiter with double underscore and dashes with single underscore.
			So that parent.child-with-dash becomes PARENT__CHILD_WITH_DASH
		*/
		s = strings.Replace(strings.ToLower(s), "__", ".", -1)
		s = strings.Replace(strings.ToLower(s), "_", "-", -1)
		return s
	}), nil); err != nil {
		log.WithError(err).Fatal("Could not parse flags")
	}

	// CLI Flags
	if err := fs.Parse(args); err != nil {
		log.WithError(err).Fatal("Could not parse flags")
	}
	if err := koanfInstance.Load(posflag.Provider(fs, ".", koanfInstance), nil); err != nil {
		log.WithError(err).Fatal("Could not process flags")
	}

	if err := koanfInstance.Unmarshal("", &config); err != nil {
		log.WithError(err).Fatal("Could not merge defaults with settings from environment variables")
	}
}

// ConvertHeaders takes a list of `key=value` headers and adds those trimmed to the specified header struct. It ignores
// any malformed entries.
func ConvertHeaders(headers []string, header *http.Header) {
	for _, hd := range headers {
		arr := strings.SplitN(hd, "=", 2)
		if len(arr) < 2 {
			log.WithFields(log.Fields{
				"arg":   hd,
				"error": "cannot split: missing equal sign",
			}).Warn("Could not parse header, ignoring")
			continue
		}
		key := strings.TrimSpace(arr[0])
		value := strings.TrimSpace(arr[1])
		log.WithFields(log.Fields{
			"key":   key,
			"value": value,
		}).Debug("Using header")
		header.Set(key, value)
	}
}
