package cfg

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
	"net/http"
	"os"
	"strings"
	"time"
)

// ParseConfig overrides internal config defaults with up CLI flags, environment variables and ensures basic validation.
func ParseConfig(version, commit, date string, fs *flag.FlagSet, args []string) *Configuration {
	config := NewDefaultConfig()

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s (version %s, %s, %s):\n", os.Args[0], version, commit, date)
		fs.PrintDefaults()
	}
	fs.String("bindAddr", config.BindAddr, "IP Address to bind to listen for Prometheus scrapes")
	fs.String("log.level", config.Log.Level, "Logging level")
	fs.BoolP("log.verbose", "v", config.Log.Verbose, "Shortcut for --log.level=debug")
	fs.StringSlice("symo.header", []string{},
		"List of \"key: value\" headers to append to the requests going to Fronius Symo")
	fs.StringP("symo.url", "u", config.Symo.URL, "Target URL of Fronius Symo device")
	fs.Int64("symo.timeout", int64(config.Symo.Timeout.Seconds()),
		"Timeout in seconds when collecting metrics from Fronius Symo. Should not be larger than the scrape interval")
	if err := viper.BindPFlags(fs); err != nil {
		log.WithError(err).Fatal("Could not bind flags")
	}

	if err := fs.Parse(args); err != nil {
		log.WithError(err).Fatal("Could not parse flags")
	}
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.AutomaticEnv()

	if err := viper.Unmarshal(config); err != nil {
		log.WithError(err).Fatal("Could not read config")
	}

	config.Symo.Timeout *= time.Second
	if config.Log.Verbose {
		config.Log.Level = "debug"
	}
	level, err := log.ParseLevel(config.Log.Level)
	if err != nil {
		log.WithError(err).Warn("Could not parse log level, fallback to info level")
		config.Log.Level = "info"
		log.SetLevel(log.InfoLevel)
	} else {
		log.SetLevel(level)
	}
	log.WithField("config", *config).Debug("Parsed config")
	return config
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
