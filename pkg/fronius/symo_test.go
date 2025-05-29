package fronius

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Symo_GetPowerFlowData_GivenUrl_WhenRequestData_ThenParseStruct(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		payload, err := os.ReadFile("testdata/example_1.json")
		require.NoError(t, err)
		_, _ = rw.Write(payload)
	}))

	c, err := NewSymoClient(ClientOptions{
		URL: server.URL,
	})
	require.NoError(t, err)

	p, err := c.GetPowerFlowData()
	assert.NoError(t, err)
	assert.Equal(t, 611.39999999999998, p.Site.PowerGrid)
	assert.Equal(t, -611.39999999999998, p.Site.PowerLoad)
	assert.Equal(t, float64(0), p.Site.PowerPhotovoltaic)
	assert.Equal(t, float64(0), p.Site.PowerAccu)
	assert.Equal(t, float64(0), p.Site.RelativeSelfConsumption)
	assert.Equal(t, 46.564, p.Site.RelativeAutonomy)
	assert.Equal(t, float64(22997), p.Site.EnergyDay)
	assert.Equal(t, float64(43059100), p.Site.EnergyTotal)
	assert.Equal(t, 3525577.75, p.Site.EnergyYear)

	assert.Equal(t, 34.5, p.Inverters["1"].BatterySoC)
}

func Test_Symo_GetArchiveData_GivenUrl_WhenRequestData_ThenParseStruct(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		payload, err := os.ReadFile("testdata/test_archive_data.json")
		require.NoError(t, err)
		_, _ = rw.Write(payload)
	}))

	c, err := NewSymoClient(ClientOptions{
		URL:                     server.URL,
		PowerFlowEnabled:        true,
		ArchiveEnabled:          true,
		InverterRealtimeEnabled: true,
	})
	require.NoError(t, err)

	p, err := c.GetArchiveData()
	assert.NoError(t, err)
	assert.Equal(t, float64(13), p["inverter/1"].Data.CurrentDCString1.Values["0"])
	assert.Equal(t, float64(15.92), p["inverter/1"].Data.CurrentDCString2.Values["0"])
	assert.Equal(t, float64(425.6), p["inverter/1"].Data.VoltageDCString1.Values["0"])
	assert.Equal(t, float64(408.90000000000003), p["inverter/1"].Data.VoltageDCString2.Values["0"])
}

func Test_Symo_GetInverterRealtimeData_GivenUrl_WhenRequestData_ThenParseStruct(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		payload, err := os.ReadFile("testdata/realtimedata.json")
		require.NoError(t, err)
		_, _ = rw.Write(payload)
	}))

	c, err := NewSymoClient(ClientOptions{
		URL:                     server.URL,
		PowerFlowEnabled:        true,
		ArchiveEnabled:          true,
		InverterRealtimeEnabled: true,
	})
	require.NoError(t, err)

	p, err := c.GetInverterRealtimeData()
	assert.NoError(t, err)

	//current
	assert.Equal(t, float64(0.021116470918059349), p.DcCurrentMPPT1.Value)
	assert.Equal(t, float64(0.01560344360768795), p.DcCurrentMPPT2.Value)
	assert.Equal(t, float64(0), p.DcCurrentMPPT3.Value)
	assert.Equal(t, float64(0), p.DcCurrentMPPT4.Value)
	assert.Equal(t, "A", p.DcCurrentMPPT1.Unit)
	assert.Equal(t, "A", p.DcCurrentMPPT2.Unit)
	assert.Equal(t, "A", p.DcCurrentMPPT3.Unit)
	assert.Equal(t, "A", p.DcCurrentMPPT4.Unit)

	//voltage
	assert.Equal(t, float64(44.587142944335938), p.DcVoltageMPPT1.Value)
	assert.Equal(t, float64(72.194984436035156), p.DcVoltageMPPT2.Value)
	assert.Equal(t, float64(0), p.DcVoltageMPPT3.Value)
	assert.Equal(t, float64(0), p.DcVoltageMPPT4.Value)
	assert.Equal(t, "V", p.DcVoltageMPPT1.Unit)
	assert.Equal(t, "V", p.DcVoltageMPPT2.Unit)
	assert.Equal(t, "V", p.DcVoltageMPPT3.Unit)
	assert.Equal(t, "V", p.DcVoltageMPPT4.Unit)

	//AC frequency
	assert.Equal(t, float64(50.029872894287109), p.AcFrequency.Value)

	//AC power
	assert.Equal(t, float64(253.71487426757812), p.AcPower.Value)

	//Total energy generated
	assert.Equal(t, float64(1392623.8052777778), p.TotalEnergyGenerated.Value)
}

func Test_Symo_GetMeterRealtimeData_GivenUrl_WhenRequestData_ThenParseStruct(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		payload, err := os.ReadFile("testdata/meterrealtimedata.json")
		require.NoError(t, err)
		_, _ = rw.Write(payload)
	}))

	c, err := NewSymoClient(ClientOptions{
		URL:                  server.URL,
		MeterRealtimeEnabled: true,
	})
	require.NoError(t, err)

	p, err := c.GetMeterRealtimeData()
	assert.NoError(t, err)
	assert.NotNil(t, p)
	assert.Equal(t, float64(12345.67), p.EnergyReal_WAC_Sum_Produced)
	assert.Equal(t, float64(7654.32), p.EnergyReal_WAC_Sum_Consumed)
}
