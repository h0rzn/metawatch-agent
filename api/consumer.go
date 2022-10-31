package api

import (
	"github.com/h0rzn/monitoring_agent/dock"
)

type Consumer struct {
	Controller *dock.Controller
}

func CreateConsumer() (*Consumer, error) {
	controller, err := dock.NewController()
	if err != nil {
		return &Consumer{}, err
	}
	return &Consumer{
		Controller: controller,
	}, nil
}

func (c *Consumer) Run() {
	c.Controller.Init()
}
