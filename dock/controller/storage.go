package controller

import (
	"context"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/h0rzn/monitoring_agent/dock/container"
	"github.com/h0rzn/monitoring_agent/dock/controller/db"
	"github.com/sirupsen/logrus"
)

// Storage stores container instances and manages changes
type Storage struct {
	mutex      sync.Mutex
	Containers map[*container.Container]*Link
	c          *client.Client
	DB         *db.DB
}

func NewStorage(c *client.Client) *Storage {
	return &Storage{
		mutex:      sync.Mutex{},
		Containers: make(map[*container.Container]*Link),
		c:          c,
		DB:         &db.DB{},
	}
}

func (s *Storage) Discover() ([]types.Container, error) {
	ctx := context.Background()
	containers, err := s.c.ContainerList(
		ctx,
		types.ContainerListOptions{
			Filters: filters.Args{},
		})
	logrus.Infof("- STORAGE - discovered %d container(s)\n", len(containers))
	return containers, err
}

func (s *Storage) Add(raw ...types.Container) error {
	var added int
	for _, rawCont := range raw {
		cont := container.NewContainer(rawCont, s.c)
		err := cont.Start()
		if err != nil {
			logrus.Errorf("- STORAGE - container failed to start (ignore): %s", err)
			continue
		}

		l := NewLink(cont.ID)

		s.mutex.Lock()
		s.Containers[cont] = l
		l.Init(cont)
		go l.Run()
		s.mutex.Unlock()
		added = added + 1
	}
	logrus.Infof("- STORAGE - added %d container(s)\n", added)
	return nil
}

func (s *Storage) Container(id string) (*container.Container, bool) {
	for container := range s.Containers {
		if container.ID == id {
			return container, true
		}
	}
	return &container.Container{}, false
}

func (s *Storage) JSONSkel() []*container.ContainerJSON {
	var skels []*container.ContainerJSON

	for container := range s.Containers {
		skels = append(skels, container.JSONSkel())
	}
	return skels
}

func (s *Storage) Links() {
	for {
		data := []interface{}{}
		for _, link := range s.Containers {
			dbSet := <-link.Out
			data = append(data, dbSet)
		}
		logrus.Infof("- STORAGE - sending %d collected sets\n", len(data))

		s.DB.Client.BulkWrite(data)
		// write data to db instance

	}
}
