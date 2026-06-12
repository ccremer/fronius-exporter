package main

import (
	"net/http"
	"os"
	"time"

	"github.com/ccremer/fronius-exporter/cfg"
	"github.com/ccremer/fronius-exporter/pkg/fronius"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
)

var (
	version     = "unknown"
	commit      = "dirty"
	date        = time.Now().String()
	config      = cfg.ParseConfig(version, commit, date, flag.NewFlagSet("main", flag.ExitOnError), os.Args[1:])
	promHandler = promhttp.Handler()
	mqttPub     *mqttPublisher
)

func main() {
	log.WithFields(log.Fields{
		"version": version,
		"commit":  commit,
		"date":    date,
	}).Info("Starting exporter.")

	headers := http.Header{}
	cfg.ConvertHeaders(config.Symo.Headers, &headers)
	symoClient, err := fronius.NewSymoClient(fronius.ClientOptions{
		URL:                     config.Symo.URL,
		Headers:                 headers,
		Timeout:                 config.Symo.Timeout,
		PowerFlowEnabled:        config.Symo.PowerFlowEnabled,
		ArchiveEnabled:          config.Symo.ArchiveEnabled,
		InverterRealtimeEnabled: config.Symo.InverterRealtimeEnabled,
		MeterRealtimeEnabled:    config.Symo.MeterRealtimeEnabled,
	})
	if err != nil {
		log.WithError(err).Fatal("Cannot initialize Fronius Symo client.")
	}
	if !config.Symo.ArchiveEnabled && !config.Symo.PowerFlowEnabled && !config.Symo.InverterRealtimeEnabled && !config.Symo.MeterRealtimeEnabled {
		log.Fatal("All scrape endpoints are disabled. You need enable at least one endpoint.")
	}

	mqttPub, err = newMQTTPublisher(config.MQTT, config.Symo, config.Poll)
	if err != nil {
		log.WithError(err).Fatal("Cannot initialize MQTT publisher.")
	}
	if mqttPub != nil {
		log.WithFields(log.Fields{
			"broker":     config.MQTT.Broker,
			"base_topic": config.MQTT.BaseTopic,
		}).Info("MQTT publishing enabled.")
	}

	state := newExporterState()
	go startPollingLoop(symoClient, state, config.Poll.Interval)
	log.WithFields(log.Fields{
		"interval":      config.Poll.Interval,
		"fresh_timeout": config.Poll.FreshTimeout,
	}).Info("Background polling enabled.")

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		log.WithFields(log.Fields{
			"uri":    r.RequestURI,
			"client": r.RemoteAddr,
		}).Debug("Accessed Root endpoint")
		http.Redirect(w, r, "/metrics", http.StatusMovedPermanently)
	})
	http.HandleFunc("/liveness", func(w http.ResponseWriter, r *http.Request) {
		log.WithFields(log.Fields{
			"uri":    r.RequestURI,
			"client": r.RemoteAddr,
		}).Debug("Accessed Liveness endpoint")
		w.WriteHeader(http.StatusNoContent)
	})
	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		log.WithFields(log.Fields{
			"uri":    r.RequestURI,
			"client": r.RemoteAddr,
		}).Debug("Accessed Metrics endpoint")
		if !state.IsFresh(symoClient.Options, config.Poll.FreshTimeout) {
			http.Error(w, "cached metrics are stale", http.StatusServiceUnavailable)
			return
		}
		promHandler.ServeHTTP(w, r)
	})

	log.WithField("port", config.BindAddr).Info("Listening for scrapes.")
	log.WithError(http.ListenAndServe(config.BindAddr, nil)).Fatal("Shutting down.")
}
