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
		logrus.Errorln(err)
		return
	}
	err = api.RegRoutes()
	if err != nil {
		logrus.Errorf("- API - JWT init err: %s\n", err)
		logrus.Errorln("This error is fatal. exit.")
		return
	}
	api.Run()
}
