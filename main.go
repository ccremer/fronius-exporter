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
		URL:              config.Symo.URL,
		Headers:          headers,
		Timeout:          config.Symo.Timeout,
		PowerFlowEnabled: config.Symo.PowerFlowEnabled,
		ArchiveEnabled:   config.Symo.ArchiveEnabled,
	})
	if err != nil {
		log.WithError(err).Fatal("Cannot initialize Fronius Symo client.")
	}

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
		collectMetricsFromTarget(symoClient)
		promHandler.ServeHTTP(w, r)
	})

	log.WithField("port", config.BindAddr).Info("Listening for scrapes.")
	log.WithError(http.ListenAndServe(config.BindAddr, nil)).Fatal("Shutting down.")
}
