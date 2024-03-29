package main

import (
	"github.com/h0rzn/monitoring_agent/api"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

func main() {
	logrus.SetLevel(logrus.DebugLevel)
	err := godotenv.Load(".env")
	if err != nil {
		logrus.Errorf("- MAIN - failed to load .env")
		return
	}
	logrus.Infoln("starting metawach-agent")
	api, err := api.NewAPI()
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
