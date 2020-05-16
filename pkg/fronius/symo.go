package fronius

import (
	"encoding/json"
	"net/http"
	"net/url"
	"time"
)

type (
	symoPowerFlow struct {
		Body struct {
			Data SymoData
		}
	}
	SymoData struct {
		Inverters map[string]Inverter
		Site      struct {
			Mode          string `json:"Mode"`
			MeterLocation string `json:"Meter_Location"`
			// PowerGrid is the power supplied by the grid in Watt. A negative value means that excess power is provided
			// back to the grid.
			PowerGrid float64 `json:"P_Grid"`
			// PowerLoad is the current load in Watt.
			PowerLoad float64 `json:"P_Load"`
			// PowerAccu is the current power supplied from Accumulator in Watt.
			PowerAccu float64 `json:"P_Akku"`
			// PowerPhotovoltaic is the current power coming from Photovoltaic in Watt.
			PowerPhotovoltaic float64 `json:"P_PV"`
			// RelativeSelfConsumption indicates the ratio between the current power generated and the
			// current load. When it reaches 1, the RelativeAutonomy declines, since the site can not produce enough
			// energy and needs support from the grid.
			RelativeSelfConsumption float64 `json:"rel_SelfConsumption"`
			// RelativeAutonomy is the ratio of how autonomous the installation is. An autonomy of 1 means that
			// the site is producing more energy than it is needed.
			RelativeAutonomy float64 `json:"rel_Autonomy"`
			// EnergyDay is the accumulated energy in kWh generated in this day so far. It is reset at the device's
			// configured timezone at midnight.
			EnergyDay float64 `json:"E_Day"`
			// EnergyYear is the accumulated energy in kWh generated in this year so far. It is reset at the device's
			// configured timezone at midnight of 31st of December.
			EnergyYear float64 `json:"E_Year"`
			// EnergyYear is the accumulated energy in kWh generated in this site so far.
			EnergyTotal float64 `json:"E_Total"`
		}
	}
	Inverter struct {
		DT          float64 `json:"DT"`
		Power       float64 `json:"P"`
		EnergyDay   float64 `json:"E_Day"`
		EnergyYear  float64 `json:"E_Year"`
		EnergyTotal float64 `json:"E_Total"`
	}
	SymoClient struct {
		request *http.Request
		Options ClientOptions
	}
	ClientOptions struct {
		URL     string
		Headers http.Header
		Timeout time.Duration
	}
)

func NewSymoClient(options ClientOptions) (*SymoClient, error) {
	u, err := url.Parse(options.URL)
	if err != nil {
		return nil, err
	}
	return &SymoClient{
		request: &http.Request{
			URL:    u,
			Header: options.Headers,
		},
		Options: options,
	}, nil
}

func (c *SymoClient) GetPowerFlowData() (*SymoData, error) {
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
