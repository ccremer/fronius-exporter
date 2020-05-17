package main

import (
	"fronius-exporter/cfg"
	"fronius-exporter/pkg/fronius"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"net/http"
	"os"
	"time"
)

var (
	version     = "unknown"
	commit      = "dirty"
	date        = time.Now().String()
	config      = cfg.ParseConfig(version, commit, date, flag.NewFlagSet("main", flag.ExitOnError), os.Args[1:])
	promHandler = promhttp.Handler()
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
		URL:     config.Symo.Url,
		Headers: headers,
		Timeout: config.Symo.Timeout,
	})
	if err != nil {
		log.WithError(err).Fatal("Cannot initialize Fronius Symo client.")
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/metrics", http.StatusMovedPermanently)
	})
	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		collectMetricsFromTarget(symoClient)
		promHandler.ServeHTTP(w, r)
	})

	log.WithField("port", config.BindAddr).Info("Listening for scrapes.")
	log.WithError(http.ListenAndServe(config.BindAddr, nil)).Fatal("Shutting down.")
}
