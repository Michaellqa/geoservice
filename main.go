package main

import (
	"geoservice/cli"
	"geoservice/geo"
	"geoservice/messaging"
)

func main() {
	broker := messaging.NewBroker()

	srvCount := 5
	services := make([]geo.Service, srvCount)
	for i := 0; i < 5; i++ {
		services[i] = geo.NewService(broker)
		services[i].Start()
	}

	cmd := cli.NewCommander(broker)
	cmd.Start()
}
