package container

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
)

type Storage struct {
	mutex      sync.Mutex
	c          *client.Client
	Containers map[*Container]bool
	Feed       chan interface{}
}

func NewStorage(c *client.Client) *Storage {
	return &Storage{
		mutex:      sync.Mutex{},
		c:          c,
		Containers: map[*Container]bool{},
		Feed:       make(chan interface{}),
	}
}

func (s *Storage) Init() error {
	ctx := context.Background()
	raws, err := s.c.ContainerList(
		ctx,
		types.ContainerListOptions{
			All: true,
		},
	)
	if err != nil {
		return err
	}
	logrus.Infof("- STORAGE - discovered %d container(s)\n", len(raws))

	for idx := range raws {
		err := s.Add(raws[idx].ID)
		if err != nil {
			logrus.Errorf("- STORAGE - failed to add container: %s\n", err)
			continue
		}
	}
	return nil
}

func (s *Storage) Add(id string) (err error) {
	if container, exists := s.Container(id); exists {
		// stopped
		if !s.Containers[container] {
			err = container.Start()
			if err != nil {
				return
			}
			s.Containers[container] = true
			return
		}
		// dont do anything if container is already running
		return
	}

	// add unindexed container
	s.mutex.Lock()
	container := NewContainer(s.c, id, s.Feed)
	err = container.Start()
	if err != nil {
		return
	}

	if container.State.Status == "running" {
		s.Containers[container] = true
		go container.RunFeed()
	} else {
		s.Containers[container] = false
	}
	s.mutex.Unlock()

	logrus.Infof("- STORAGE - added %s container\n", container.State.Status)

	return
}

func (s *Storage) Register(id string) (err error) {

	return
}

func (s *Storage) Stop(id string) error {
	s.mutex.Lock()
	if container, exists := s.Container(id); exists {
		running := s.Containers[container]
		if running {
			err := container.Stop()
			if err != nil {
				return err
			}
			s.Containers[container] = false
		}
	}
	s.mutex.Unlock()
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
		for item := range s.Feed {
			data = append(data, item)

			select {
			case <-sendTick.C:
				if len(data) > 0 {
					out <- data
					data = nil
				}
			default:
			}
		}
		close(out)
	}()
	return out
}
