package container

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
)

type Storage struct {
	mutex      sync.Mutex
	c          *client.Client
	Containers map[*Container]bool
	feed       chan interface{}
}

func NewStorage(c *client.Client) *Storage {
	return &Storage{
		mutex:      sync.Mutex{},
		c:          c,
		Containers: map[*Container]bool{},
		feed:       make(chan interface{}),
	}
}

func (s *Storage) Init() error {
	ctx := context.Background()
	raws, err := s.c.ContainerList(
		ctx,
		types.ContainerListOptions{
			Filters: filters.Args{},
		},
	)
	if err != nil {
		return err
	}
	logrus.Infof("- STORAGE - discovered %d container(s)\n", len(raws))

	for idx := range raws {
		s.AddRaw(raws[idx])
	}

	return nil
}

func (s *Storage) AddRaw(raw types.Container) error {
	s.mutex.Lock()
	container := NewContainer(raw, s.c, s.feed)
	s.Containers[container] = true
	err := container.Start()
	if err != nil {
		logrus.Errorf("- STORAGE - failed to add raw container: %s\n", err)
	}
	s.mutex.Unlock()
	return nil
}

func (s *Storage) Add(id string) error {
	s.mutex.Lock()
	if _, exists := s.Container(id); exists {
		return nil
	}
	ctx := context.Background()
	idFilter := filters.NewArgs()
	idFilter.Add("id", id)
	containers, err := s.c.ContainerList(
		ctx,
		types.ContainerListOptions{
			Filters: idFilter,
		})
	if err != nil {
		return err
	}
	raw := containers[0]
	err = s.AddRaw(raw)
	if err != nil {
		return err
	}

	s.mutex.Unlock()
	logrus.Infoln("- STORAGE - added container")
	return nil
}

func (s *Storage) Remove(id string) error {
	s.mutex.Lock()
	if container, exists := s.Container(id); exists {
		fmt.Println("attempt stop")
		err := container.Stop()
		fmt.Println("container stopped")
		if err != nil {
			return err
		}
		delete(s.Containers, container)
		logrus.Infof("- STORAGE - container removed: %d left\n", len(s.Containers))
	} else {
		logrus.Warningln("- STORAGE - tried to remove unkown container")
	}
	s.mutex.Unlock()

	return nil
}

func (s *Storage) Container(id string) (*Container, bool) {
	for container := range s.Containers {
		if container.ID == id {
			return container, true
		}
	}
	return &Container{}, false
}

func (s *Storage) Items() (containers []*Container) {
	s.mutex.Lock()
	for container := range s.Containers {
		containers = append(containers, container)
	}
	s.mutex.Unlock()
	return
}

func (s *Storage) Broadcast() chan []interface{} {
	out := make(chan []interface{})
	go func() {
		sendTick := time.NewTicker(5 * time.Second)
		data := []interface{}{}
		for {
			select {
			case item := <-s.feed:
				data = append(data, item)
			case <-sendTick.C:
				out <- data
				data = nil
			}
		}
	}()
	return out
}
