package fronius

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

func Test_Symo_GetPowerFlowData_GivenUrl_WhenRequestData_ThenParseStruct(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		payload, err := ioutil.ReadFile("testdata/example_1.json")
		require.NoError(t, err)
		rw.Write(payload)
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
	assert.Equal(t, 0.5, p.Site.RelativeSelfConsumption)
	assert.Equal(t, float64(1), p.Site.RelativeAutonomy)
	assert.Equal(t, float64(22997), p.Site.EnergyDay)
	assert.Equal(t, float64(43059100), p.Site.EnergyTotal)
	assert.Equal(t, 3525577.75, p.Site.EnergyYear)
}
