package fronius

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

const (
	// PowerDataPath is the Fronius API URL-path for power real time data
	PowerDataPath = "/solar_api/v1/GetPowerFlowRealtimeData.fcgi"
	// ArchiveDataPath is the Fronius API URL-path for archive data
	ArchiveDataPath = "/solar_api/v1/GetArchiveData.cgi?Scope=System&Channel=Voltage_DC_String_1&Channel=Current_DC_String_1&Channel=Voltage_DC_String_2&Channel=Current_DC_String_2&HumanReadable=false"
)

type (
	symoPowerFlow struct {
		Body struct {
			Data SymoData
		}
	}
	// SymoData holds the parsed data from the Symo API.
	SymoData struct {
		Inverters map[string]Inverter
		Site      struct {
			Mode          string `json:"Mode"`
			MeterLocation string `json:"Meter_Location"`
			// PowerGrid is the power supplied by the grid in Watt.
			// A negative value means that excess power is provided back to the grid.
			PowerGrid float64 `json:"P_Grid"`
			// PowerLoad is the current load in Watt.
			PowerLoad float64 `json:"P_Load"`
			// PowerAccu is the current power supplied from Accumulator in Watt.
			PowerAccu float64 `json:"P_Akku"`
			// PowerPhotovoltaic is the current power coming from Photovoltaic in Watt.
			PowerPhotovoltaic float64 `json:"P_PV"`
			// RelativeSelfConsumption indicates the ratio between the current power generated and the current load.
			// When it reaches 100, the RelativeAutonomy declines, since the site can not produce enough energy and needs support from the grid.
			// If the device returns null in PowerPhotovoltaic, this field becomes also 0!
			RelativeSelfConsumption float64 `json:"rel_SelfConsumption"`
			// RelativeAutonomy is the ratio of how autonomous the installation is.
			// An autonomy of 100 means that the site is producing more energy than it is needed.
			RelativeAutonomy float64 `json:"rel_Autonomy"`
			// EnergyDay is the accumulated energy in Wh generated in this day so far.
			// It is reset at the device's configured timezone at midnight.
			EnergyDay float64 `json:"E_Day"`
			// EnergyYear is the accumulated energy in Wh generated in this year so far.
			// It is reset at the device's configured timezone at midnight of 31st of December.
			EnergyYear float64 `json:"E_Year"`
			// EnergyTotal is the accumulated energy in Wh generated in this site so far.
			EnergyTotal float64 `json:"E_Total"`
		}
	}
	// Inverter represents a power inverter installed at the Fronius Symo site.
	Inverter struct {
		DT          float64 `json:"DT"`
		Power       float64 `json:"P"`
		BatterySoC  float64 `json:"SOC"`
		EnergyDay   float64 `json:"E_Day"`
		EnergyYear  float64 `json:"E_Year"`
		EnergyTotal float64 `json:"E_Total"`
	}

	// SymoArchive holds the parsed archive data from Symo API
	symoArchive struct {
		Body struct {
			Data map[string]InverterArchive
		}
	}

	// InverterArchive represents a power archive data with its channels
	InverterArchive struct {
		Data struct {
			CurrentDCString1 Channel `json:"Current_DC_String_1"`
			CurrentDCString2 Channel `json:"Current_DC_String_2"`
			VoltageDCString1 Channel `json:"Voltage_DC_String_1"`
			VoltageDCString2 Channel `json:"Voltage_DC_String_2"`
		}
	}

	// Channel represents the inverter channel data
	Channel struct {
		Unit   string
		Values map[string]float64
	}

	// SymoClient is a wrapper for making API requests against a Fronius Symo device.
	SymoClient struct {
		request *http.Request
		Options ClientOptions
	}
	// ClientOptions holds some parameters for the SymoClient.
	ClientOptions struct {
		URL              string
		Headers          http.Header
		Timeout          time.Duration
		PowerFlowEnabled bool
		ArchiveEnabled   bool
	}
)

// NewSymoClient constructs a SymoClient ready to use for collecting metrics.
func NewSymoClient(options ClientOptions) (*SymoClient, error) {
	return &SymoClient{
		request: &http.Request{
			Header: options.Headers,
		},
		Options: options,
	}, nil
}

// GetPowerFlowData returns the parsed data from the Symo device.
func (c *SymoClient) GetPowerFlowData() (*SymoData, error) {
	url, err := url.Parse(c.Options.URL + PowerDataPath)

	if err != nil {
		return nil, err
	}

	c.request.URL = url
	client := http.DefaultClient
	client.Timeout = c.Options.Timeout
	response, err := client.Do(c.request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	p := symoPowerFlow{}
	err = json.NewDecoder(response.Body).Decode(&p)
	if err != nil {
		return nil, err
	}
	return &p.Body.Data, nil
}

// GetArchiveData returns the parsed data from the Symo device.
func (c *SymoClient) GetArchiveData() (map[string]InverterArchive, error) {
	url, err := url.Parse(c.Options.URL + ArchiveDataPath)

	if err != nil {
		return nil, err
	}

	c.request.URL = url
	client := http.DefaultClient
	client.Timeout = c.Options.Timeout
	q := c.request.URL.Query()
	q.Del("StartDate")
	q.Del("EndDate")

	c.request.URL.RawQuery = fmt.Sprintf("%s&StartDate=%s&EndDate=%s",
		q.Encode(),
		time.Now().Truncate(5*time.Minute).UTC().Local().Format(time.RFC3339),
		time.Now().Add(5*time.Minute).Truncate(5*time.Minute).UTC().Local().Format(time.RFC3339))

	response, err := client.Do(c.request)

	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	p := symoArchive{}
	err = json.NewDecoder(response.Body).Decode(&p)
	if err != nil {
		return nil, err
	}
	return p.Body.Data, nil
}
