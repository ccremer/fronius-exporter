package cfg

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
	"net/http"
	"os"
	"strings"
)

func ParseConfig(version, commit, date string) *Configuration {
	config := NewDefaultConfig()

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s (version %s, %s, %s):\n", os.Args[0], version, commit, date)
		flag.PrintDefaults()
	}
	flag.String("bindAddr", config.BindAddr, "IP Address to bind to listen for Prometheus scrapes")
	flag.String("log.level", config.Log.Level, "Logging level")
	flag.BoolP("log.verbose", "v", config.Log.Verbose, "Shortcut for --log.level=debug")
	flag.StringSlice("symo.headers", []string{},
		"List of \"key: value\" headers to append to the requests going to Fronius Symo")
	flag.String("symo.url", config.Symo.Url, "Target URL of Fronius Symo device")
	if err := viper.BindPFlags(flag.CommandLine); err != nil {
		log.WithError(err).Fatal(err)
	}

	flag.Parse()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.AutomaticEnv()

	if err := viper.Unmarshal(config); err != nil {
		log.WithError(err).Fatal("Could not read config")
	}

	if config.Log.Verbose {
		config.Log.Level = "debug"
	}
	level, err := log.ParseLevel(config.Log.Level)
	if err != nil {
		log.WithError(err).Warn("Could not parse log level, fallback to info level")
		log.SetLevel(log.InfoLevel)
	} else {
		log.SetLevel(level)
	}
	log.WithField("config", *config).Debug("Parsed config.")
	return config
}

func ConvertHeaders(headers []string, header *http.Header) {
	for _, hd := range headers {
		arr := strings.SplitN(hd, ":", 2)
		if len(arr) < 2 {
			log.WithFields(log.Fields{
				"arg":   hd,
				"error": "cannot split: missing colon",
			}).Warn("Could not parse header, ignoring")
			continue
		}
		key := strings.TrimSpace(arr[0])
		value := strings.TrimSpace(arr[1])
		log.WithFields(log.Fields{
			"key":   key,
			"value": value,
		}).Debug("Using header.")
		header.Set(key, value)
	}
}
