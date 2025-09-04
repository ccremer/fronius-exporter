package main

import (
	"strings"
	"sync"
	"time"

	"github.com/ccremer/fronius-exporter/pkg/fronius"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	log "github.com/sirupsen/logrus"
)

var (
	namespace           = "fronius"
	scrapeDurationGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "scrape_duration_seconds",
		Help:      "Time it took to scrape the device in seconds",
	})
	scrapeErrorCount = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "scrape_error_count",
		Help:      "Number of scrape errors",
	})

	inverterPowerGaugeVec = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "inverter_power",
		Help:      "Power flow of the inverter in Watt",
	}, []string{"inverter"})
	inverterBatteryChargeGaugeVec = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "inverter_soc",
		Help:      "State of charge of the battery attached to the inverter in percent",
	}, []string{"inverter"})

	sitePowerLoadGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "site_power_load",
		Help:      "Site power load in Watt",
	})
	sitePowerGridGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "site_power_grid",
		Help:      "Site power supplied to or provided from the grid in Watt",
	})
	sitePowerAccuGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "site_power_accu",
		Help:      "Site power supplied to or provided from the accumulator(s) in Watt",
	})
	sitePowerPhotovoltaicsGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "site_power_photovoltaic",
		Help:      "Site power supplied to or provided from the accumulator(s) in Watt",
	})

	siteAutonomyRatioGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "site_autonomy_ratio",
		Help:      "Relative autonomy ratio of the site",
	})
	siteSelfConsumptionRatioGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "site_selfconsumption_ratio",
		Help:      "Relative self consumption ratio of the site",
	})

	siteEnergyGaugeVec = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "site_energy_consumption",
		Help:      "Energy consumption in kWh",
	}, []string{"time_frame"})

	siteMPPTVoltageGaugeVec = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "site_mppt_voltage",
		Help:      "Site mppt voltage in V",
	}, []string{"inverter", "mppt"})

	siteMPPTCurrentDCGaugeVec = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "site_mppt_current_dc",
		Help:      "Site mppt current DC in A",
	}, []string{"inverter", "mppt"})

	siteRealtimeDataDcCurrentMPPT1Gauge = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "site_realtime_data_dc_current_mppt1",
		Help:      "Site real time data DC current MPPT 1 in A",
	})
	siteRealtimeDataDcCurrentMPPT2Gauge = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "site_realtime_data_dc_current_mppt2",
		Help:      "Site real time data DC current MPPT 2 in A",
	})
	siteRealtimeDataDcCurrentMPPT3Gauge = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "site_realtime_data_dc_current_mppt3",
		Help:      "Site real time data DC current MPPT 3 in A",
	})
	siteRealtimeDataDcCurrentMPPT4Gauge = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "site_realtime_data_dc_current_mppt4",
		Help:      "Site real time data DC current MPPT 4 in A",
	})
	siteRealtimeDataDcVoltageMPPT1Gauge = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "site_realtime_data_dc_voltage_mppt1",
		Help:      "Site real time data DC voltage MPPT 1 in V",
	})
	siteRealtimeDataDcVoltageMPPT2Gauge = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "site_realtime_data_dc_voltage_mppt2",
		Help:      "Site real time data DC voltage MPPT 2 in V",
	})
	siteRealtimeDataDcVoltageMPPT3Gauge = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "site_realtime_data_dc_voltage_mppt3",
		Help:      "Site real time data DC voltage MPPT 3 in V",
	})
	siteRealtimeDataDcVoltageMPPT4Gauge = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "site_realtime_data_dc_voltage_mppt4",
		Help:      "Site real time data DC voltage MPPT 4 in V",
	})
	siteRealtimeDataAcFrequencyGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "site_realtime_data_ac_frequency",
		Help:      "Site real time data AC frequency in Hz",
	})
	siteRealtimeDataAcPowerGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "site_realtime_data_ac_power",
		Help:      "Site real time data AC power in W",
	})
	siteRealtimeDataTotalEnergyGeneratedGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "site_realtime_data_total_energy_generated",
		Help:      "Site real time data total energy generated in Wh",
	})
	siteMeterRealTimeDataEnergyReal_WAC_Sum_Produced = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "site_meter_real_time_data_energy_real_wac_sum_produced",
		Help:      "Site meter real time data energy real WAC sum produced in Wh",
	})
	siteMeterRealTimeDataEnergyReal_WAC_Sum_Consumed = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "site_meter_real_time_data_energy_real_wac_sum_consumed",
		Help:      "Site meter real time data energy real WAC sum consumed in Wh",
	})
)

func collectMetricsFromTarget(client *fronius.SymoClient) {
	start := time.Now()
	log.WithFields(log.Fields{
		"url":              client.Options.URL,
		"timeout":          client.Options.Timeout,
		"powerFlowEnabled": client.Options.PowerFlowEnabled,
		"archiveEnabled":   client.Options.ArchiveEnabled,
		"inverterRealtime": client.Options.InverterRealtimeEnabled,
		"meterRealtime":    client.Options.MeterRealtimeEnabled,
	}).Debug("Requesting data.")

	wg := sync.WaitGroup{}
	wg.Add(4)

	collectPowerFlowData(client, &wg)
	collectArchiveData(client, &wg)
	collectInverterRealtimeData(client, &wg)
	collectMeterRealtimeData(client, &wg)

	wg.Wait()
	elapsed := time.Since(start)
	scrapeDurationGauge.Set(elapsed.Seconds())
}

func collectPowerFlowData(client *fronius.SymoClient, w *sync.WaitGroup) {
	defer w.Done()
	if client.Options.PowerFlowEnabled {
		powerFlowData, err := client.GetPowerFlowData()
		if err != nil {
			log.WithError(err).Warn("Could not collect Symo power metrics.")
			scrapeErrorCount.Add(1)
			return
		}
		parsePowerFlowMetrics(powerFlowData)
	}
}

func collectInverterRealtimeData(client *fronius.SymoClient, w *sync.WaitGroup) {
	defer w.Done()
	if client.Options.InverterRealtimeEnabled {
		powerFlowData, err := client.GetInverterRealtimeData()
		if err != nil {
			log.WithError(err).Warn("Could not collect Symo inverter realtime metrics.")
			scrapeErrorCount.Add(1)
			return
		}
		parseInverterRealtimeData(powerFlowData)
	}
}

func collectMeterRealtimeData(client *fronius.SymoClient, w *sync.WaitGroup) {
	defer w.Done()
	if client.Options.MeterRealtimeEnabled {
		meterData, err := client.GetMeterRealtimeData()
		if err != nil {
			log.WithError(err).Warn("Could not collect Symo meter realtime metrics.")
			scrapeErrorCount.Add(1)
			return
		}
		parseMeterRealtimeData(meterData)
	}
}

func collectArchiveData(client *fronius.SymoClient, w *sync.WaitGroup) {
	defer w.Done()
	if client.Options.ArchiveEnabled {
		archiveData, err := client.GetArchiveData()
		if err != nil {
			log.WithError(err).Warn("Could not collect Symo archive metrics.")
			scrapeErrorCount.Add(1)
			return
		}
		parseArchiveMetrics(archiveData)
	}
}

func parsePowerFlowMetrics(data *fronius.SymoData) {
	log.WithField("powerFlowData", *data).Debug("Parsing data.")
	for key, inverter := range data.Inverters {
		inverterPowerGaugeVec.WithLabelValues(key).Set(inverter.Power)
		inverterBatteryChargeGaugeVec.WithLabelValues(key).Set(inverter.BatterySoC / 100)
	}
	sitePowerAccuGauge.Set(data.Site.PowerAccu)
	sitePowerGridGauge.Set(data.Site.PowerGrid)
	sitePowerLoadGauge.Set(data.Site.PowerLoad)
	sitePowerPhotovoltaicsGauge.Set(data.Site.PowerPhotovoltaic)

	siteEnergyGaugeVec.WithLabelValues("day").Set(data.Site.EnergyDay)
	siteEnergyGaugeVec.WithLabelValues("year").Set(data.Site.EnergyYear)
	siteEnergyGaugeVec.WithLabelValues("total").Set(data.Site.EnergyTotal)

	siteAutonomyRatioGauge.Set(data.Site.RelativeAutonomy / 100)
	if data.Site.PowerPhotovoltaic == 0 {
		siteSelfConsumptionRatioGauge.Set(1)
	} else {
		siteSelfConsumptionRatioGauge.Set(data.Site.RelativeSelfConsumption / 100)
	}
}

func parseInverterRealtimeData(data *fronius.SymoInverterRealtimeData) {
	log.WithField("InverterRealtimeData", *data).Debug("Parsing data.")
	siteRealtimeDataDcCurrentMPPT1Gauge.Set(data.DcCurrentMPPT1.Value)
	siteRealtimeDataDcCurrentMPPT2Gauge.Set(data.DcCurrentMPPT2.Value)
	siteRealtimeDataDcCurrentMPPT3Gauge.Set(data.DcCurrentMPPT3.Value)
	siteRealtimeDataDcCurrentMPPT4Gauge.Set(data.DcCurrentMPPT4.Value)

	siteRealtimeDataDcVoltageMPPT1Gauge.Set(data.DcVoltageMPPT1.Value)
	siteRealtimeDataDcVoltageMPPT2Gauge.Set(data.DcVoltageMPPT2.Value)
	siteRealtimeDataDcVoltageMPPT3Gauge.Set(data.DcVoltageMPPT3.Value)
	siteRealtimeDataDcVoltageMPPT4Gauge.Set(data.DcVoltageMPPT4.Value)

	siteRealtimeDataAcFrequencyGauge.Set(data.AcFrequency.Value)
	siteRealtimeDataAcPowerGauge.Set(data.AcPower.Value)
	siteRealtimeDataTotalEnergyGeneratedGauge.Set(data.TotalEnergyGenerated.Value)
}

func parseMeterRealtimeData(data *fronius.SymoMeterRealtimeData) {
	log.WithField("MeterRealtimeData", *data).Debug("Parsing data.")
	siteMeterRealTimeDataEnergyReal_WAC_Sum_Consumed.Set(data.EnergyReal_WAC_Sum_Consumed)
	siteMeterRealTimeDataEnergyReal_WAC_Sum_Produced.Set(data.EnergyReal_WAC_Sum_Produced)
}

func parseArchiveMetrics(data map[string]fronius.InverterArchive) {
	log.WithField("archiveData", data).Debug("Parsing data.")
	for key, inverter := range data {
		key = strings.TrimPrefix(key, "inverter/")
		siteMPPTCurrentDCGaugeVec.WithLabelValues(key, "1").Set(inverter.Data.CurrentDCString1.Values["0"])
		siteMPPTCurrentDCGaugeVec.WithLabelValues(key, "2").Set(inverter.Data.CurrentDCString2.Values["0"])
		siteMPPTVoltageGaugeVec.WithLabelValues(key, "1").Set(inverter.Data.VoltageDCString1.Values["0"])
		siteMPPTVoltageGaugeVec.WithLabelValues(key, "2").Set(inverter.Data.VoltageDCString2.Values["0"])
	}
}
