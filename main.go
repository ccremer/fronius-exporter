package main

import (
	"net/http"
	"os"
	"time"

	"github.com/ccremer/fronius-exporter/cfg"
	"github.com/ccremer/fronius-exporter/pkg/fronius"
	"github.com/gin-gonic/gin"
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
	symoClient  *fronius.SymoClient
)

func main() {
	var err error

	log.WithFields(log.Fields{
		"version": version,
		"commit":  commit,
		"date":    date,
	}).Info("Starting exporter.")

	gin.SetMode(gin.ReleaseMode)

	headers := http.Header{}
	cfg.ConvertHeaders(config.Symo.Headers, &headers)
	symoClient, err = fronius.NewSymoClient(fronius.ClientOptions{
		URL:              config.Symo.URL,
		Headers:          headers,
		Timeout:          config.Symo.Timeout,
		PowerFlowEnabled: config.Symo.PowerFlowEnabled,
		ArchiveEnabled:   config.Symo.ArchiveEnabled,
	})
	if err != nil {
		log.WithError(err).Fatal("Cannot initialize Fronius Symo client.")
	}
	if !config.Symo.ArchiveEnabled && !config.Symo.PowerFlowEnabled {
		log.Fatal("All scrape endpoints are disabled. You need enable at least one endpoint.")
	}

	router := gin.Default()

	rg := router.Group("/")
	rg.GET("/liveness", func(ctx *gin.Context) {
		log.WithFields(log.Fields{
			"uri":    ctx.Request.RequestURI,
			"client": ctx.Request.RemoteAddr,
		}).Debug("Accessed Liveness endpoint")
		ctx.String(http.StatusNoContent, "")
	})

	if config.BasicAuth.Username != "" {
		if config.BasicAuth.Password == "" {
			log.Fatal("Must set basic-auth.password to enable basic auth.")
		}

		rg.Use(gin.BasicAuth(gin.Accounts{
			config.BasicAuth.Username: config.BasicAuth.Password,
		}))
	}

	rg.GET("/", func(ctx *gin.Context) {
		log.WithFields(log.Fields{
			"uri":    ctx.Request.RequestURI,
			"client": ctx.Request.RemoteAddr,
		}).Debug("Accessed Root endpoint")
		ctx.Redirect(http.StatusTemporaryRedirect, "/metrics")
	})

	rg.GET("/metrics", func(ctx *gin.Context) {
		log.WithFields(log.Fields{
			"uri":    ctx.Request.RequestURI,
			"client": ctx.Request.RemoteAddr,
		}).Debug("Accessed Metrics endpoint")
		collectMetricsFromTarget(symoClient)
		promHandler.ServeHTTP(ctx.Writer, ctx.Request)
	})

	log.WithField("port", config.BindAddr).Info("Listening for scrapes.")
	log.WithError(router.Run(config.BindAddr)).Fatal("Shutting down.")
}
