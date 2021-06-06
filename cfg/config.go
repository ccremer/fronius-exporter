package cfg

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/knadh/koanf"
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
	fs.StringP("symo.url", "u", config.Symo.URL, "Target URL of Fronius Symo device.")
	fs.StringP("symo.site", "s", config.Symo.Site, "Site name of the Fronius Symo device, it's added as a static label to the metrics. Defaults to the hostname in --symo.url.")
	fs.Int64("symo.timeout", int64(config.Symo.Timeout.Seconds()),
		"Timeout in seconds when collecting metrics from Fronius Symo. Should not be larger than the scrape interval.")

}

func postLoadProcess(config *Configuration) {
	config.Symo.Timeout *= time.Second
	if config.Log.Verbose {
		config.Log.Level = "debug"
	}

	var parsedHeaders []string
	for _, header := range config.Symo.Headers {
		parsedHeaders = splitHeaderStrings(header, parsedHeaders)
	}
	config.Symo.Headers = parsedHeaders

	config.Symo.Site = getSiteOrHostName(config.Symo.Site, config.Symo.URL)

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

// getSiteOrHostName returns the site name as it was given in the config.
// If that's empty, it returns the hostname or IP address given in the Symo URL.
// If the URL cannot be parsed either, it returns empty string.
func getSiteOrHostName(site, rawURL string) string {
	if site != "" {
		return site
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return u.Host
}
