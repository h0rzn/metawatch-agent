package controller

import (
	"context"
	"fmt"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/h0rzn/monitoring_agent/dock/container"
	"github.com/h0rzn/monitoring_agent/dock/controller/db"
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
	fmt.Println("[STORAGE] discovering...")
	ctx := context.Background()
	containers, err := s.c.ContainerList(
		ctx,
		types.ContainerListOptions{
			Filters: filters.Args{},
		})
	fmt.Printf("discovered %d containers\n", len(containers))
	return containers, err
}

func (s *Storage) Add(raw ...types.Container) error {
	for _, rawCont := range raw {
		cont := container.NewContainer(rawCont, s.c)
		err := cont.Start()
		if err != nil {
			fmt.Println("[STORAGE] ignoring failed container start")
			return err
		}

		l := NewLink(cont.ID)

		s.mutex.Lock()
		s.Containers[cont] = l
		l.Init(cont)
		go l.Run()
		s.mutex.Unlock()
	}
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
		data := []db.WriteSet{}
		for _, link := range s.Containers {
			wr := <-link.Out
			data = append(data, wr)
		}
		fmt.Println("[STORAGE] sets collected, sending")

		_ = data
		// write data to db instance
	}
}
