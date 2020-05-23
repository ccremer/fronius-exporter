// +build examples

package examples

import (
	"fmt"
	"fronius-exporter/pkg/fronius"
	"log"
	"time"
)

func main() {
	client, err := fronius.NewSymoClient(fronius.ClientOptions{
		URL:     "http://symo.ip.or.hostname/solar_api/v1/GetPowerFlowRealtimeData.fcgi",
		Timeout: 5 * time.Second,
	})
	if err != nil {
		log.Fatal(err)
	}
	data, err := client.GetPowerFlowData()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Current power usage: " + data.Site.PowerLoad)
}
