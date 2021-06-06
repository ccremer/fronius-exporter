package main

import (
	"time"

	"github.com/ccremer/fronius-exporter/pkg/fronius"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	log "github.com/sirupsen/logrus"
)

var (
	namespace           = "fronius"
	scrapeDurationGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace:   namespace,
		Name:        "scrape_duration_seconds",
		ConstLabels: getConstantLabels(),
		Help:        "Time it took to scrape the device in seconds",
	})
	scrapeErrorCount = promauto.NewCounter(prometheus.CounterOpts{
		Namespace:   namespace,
		Name:        "scrape_error_count",
		ConstLabels: getConstantLabels(),
		Help:        "Number of scrape errors",
	})

	inverterPowerGaugeVec = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace:   namespace,
		Name:        "inverter_power",
		ConstLabels: getConstantLabels(),
		Help:        "Power flow of the inverter in Watt",
	}, []string{"inverter"})

	sitePowerLoadGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace:   namespace,
		Name:        "site_power_load",
		ConstLabels: getConstantLabels(),
		Help:        "Site power load in Watt",
	})
	sitePowerGridGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace:   namespace,
		Name:        "site_power_grid",
		ConstLabels: getConstantLabels(),
		Help:        "Site power supplied to or provided from the grid in Watt",
	})
	sitePowerAccuGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace:   namespace,
		Name:        "site_power_accu",
		ConstLabels: getConstantLabels(),
		Help:        "Site power supplied to or provided from the accumulator(s) in Watt",
	})
	sitePowerPhotovoltaicsGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace:   namespace,
		Name:        "site_power_photovoltaic",
		ConstLabels: getConstantLabels(),
		Help:        "Site power supplied to or provided from the accumulator(s) in Watt",
	})

	siteAutonomyRatioGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace:   namespace,
		Name:        "site_autonomy_ratio",
		ConstLabels: getConstantLabels(),
		Help:        "Relative autonomy ratio of the site",
	})
	siteSelfConsumptionRatioGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace:   namespace,
		Name:        "site_selfconsumption_ratio",
		ConstLabels: getConstantLabels(),
		Help:        "Relative self consumption ratio of the site",
	})

	siteEnergyGaugeVec = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace:   namespace,
		Name:        "site_energy_consumption",
		ConstLabels: getConstantLabels(),
		Help:        "Energy consumption in kWh",
	}, []string{"time_frame"})
)

func getConstantLabels() prometheus.Labels {
	return map[string]string{
		"site": config.Symo.Site,
	}
}

func collectMetricsFromTarget(client *fronius.SymoClient) {
	start := time.Now()
	log.WithFields(log.Fields{
		"url":     client.Options.URL,
		"timeout": client.Options.Timeout,
	}).Debug("Requesting data.")
	data, err := client.GetPowerFlowData()
	if err != nil {
		log.WithError(err).Warn("Could not collect Symo metrics.")
		scrapeErrorCount.Add(1)
	} else {
		parseMetrics(data)
	}

	elapsed := time.Since(start)
	scrapeDurationGauge.Set(elapsed.Seconds())
}

func parseMetrics(data *fronius.SymoData) {
	log.WithField("data", *data).Debug("Parsing data.")
	for key, inverter := range data.Inverters {
		inverterPowerGaugeVec.WithLabelValues(key).Set(inverter.Power)
	}
	sitePowerAccuGauge.Set(data.Site.PowerAccu)
	sitePowerGridGauge.Set(data.Site.PowerGrid)
	sitePowerLoadGauge.Set(data.Site.PowerLoad)
	sitePowerPhotovoltaicsGauge.Set(data.Site.PowerPhotovoltaic)

	siteEnergyGaugeVec.WithLabelValues("day").Set(data.Site.EnergyDay)
	siteEnergyGaugeVec.WithLabelValues("year").Set(data.Site.EnergyYear)
	siteEnergyGaugeVec.WithLabelValues("total").Set(data.Site.EnergyTotal)

	siteAutonomyRatioGauge.Set(data.Site.RelativeAutonomy)
	siteSelfConsumptionRatioGauge.Set(data.Site.RelativeSelfConsumption)
}
