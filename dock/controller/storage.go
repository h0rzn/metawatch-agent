package controller

import (
	"fmt"
	"sync"

	"github.com/docker/docker/client"
	"github.com/h0rzn/monitoring_agent/dock/container"
	"github.com/h0rzn/monitoring_agent/dock/controller/db"
	"github.com/h0rzn/monitoring_agent/dock/image"
	"github.com/sirupsen/logrus"
)

// Storage stores container instances and manages changes
// container Images are modified and passed through from docker engine api
type Storage struct {
	mutex          sync.RWMutex
	c              *client.Client
	DB             *db.DB
	Events         *Events
	ContainerStore *container.Storage
	ImageStore     *image.Storage
}

func NewStorage(c *client.Client) *Storage {
	strg := &Storage{
		mutex:          sync.RWMutex{},
		c:              c,
		DB:             &db.DB{},
		ContainerStore: container.NewStorage(c),
		ImageStore:     image.NewStorage(c),
	}
	strg.Events = NewEvents(c, strg)
	go strg.Events.Run()
	return strg
}

func (s *Storage) Init() (err error) {
	err = s.ContainerStore.Init()
	if err != nil {
		logrus.Errorf("- STORAGE - (containers) failed to init: %s\n", err)
		return
	}

	err = s.ImageStore.Init()
	if err != nil {
		logrus.Errorf("- STORAGE - (images) failed to init: %s\n", err)
		return
	}

	err = s.DB.Init()
	if err != nil {
		logrus.Errorf("- STORAGE - (db) failed to init: %s\n", err)
	}

	go func() {
		containerFeed := s.ContainerStore.Broadcast()
		for item := range containerFeed {
			fmt.Println("storage: snd bulkwrite")
			s.DB.Client.BulkWrite(item)
		}
		fmt.Println("feed writer left")
	}()

	return nil
}
