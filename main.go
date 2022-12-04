package main

import (
	"github.com/h0rzn/monitoring_agent/api"
	"github.com/sirupsen/logrus"
)

func main() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.Infoln("starting metawach-agent")
	api, err := api.NewAPI(":8080")
	if err != nil {
		panic(err)
	}
	api.RegRoutes()
	api.Run()
}
