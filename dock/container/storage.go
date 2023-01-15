package container

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/h0rzn/monitoring_agent/dock/image"
	"github.com/h0rzn/monitoring_agent/dock/metrics"
	"github.com/sirupsen/logrus"
)

type ImageGet func(string) (*image.Image, bool)

type Storage struct {
	mutex      sync.Mutex
	c          *client.Client
	Containers map[*Container]bool
	Feed       chan FeedItem
	ImageGet   ImageGet
}

func NewStorage(c *client.Client) *Storage {
	return &Storage{
		mutex:      sync.Mutex{},
		c:          c,
		Containers: map[*Container]bool{},
	}
}

func (s *Storage) Init(imgFunc ImageGet) error {
	// make image retrieval availabe for containers
	s.ImageGet = imgFunc

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
	container.ImageGet = s.ImageGet
	err = container.Start()
	if err != nil {
		return
	}

	if container.State.Status == "running" {
		s.Containers[container] = true
		go container.RunFeed()
		go container.Streams.Metrics.HandleLatest()
	} else {
		s.Containers[container] = false
	}
	s.mutex.Unlock()

	logrus.Infof("- STORAGE - added %s container\n", container.State.Status)

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

func (s *Storage) MarshalJSON() ([]byte, error) {
	containers := make([]*Container, 0)
	s.mutex.Lock()
	for c := range s.Containers {
		containers = append(containers, c)
	}
	s.mutex.Unlock()

	return json.Marshal(containers)
}

func (s *Storage) CollectLatest() (colLatest []metrics.Set) {
	s.mutex.Lock()
	fmt.Printf("store collecting latest for %d containers\n", len(s.Containers))
	for container, active := range s.Containers {
		if active {
			latest := container.Streams.Metrics.Latest()
			colLatest = append(colLatest, latest)
		}
	}
	s.mutex.Unlock()
	return
}

func (s *Storage) Broadcast() chan []interface{} {
	fmt.Println("broadcast running")
	out := make(chan []interface{})
	go func() {
		data := make([]interface{}, 0)
		ticker := time.NewTicker(5 * time.Second)
		for item := range s.Feed {
			fmt.Println("storage handling feed item")
			select {
			case <-ticker.C:
				out <- data
				fmt.Println("storage: broadcasted data")
				data = nil
			default:
			}
			data = append(data, item)
		}
		close(out)
	}()
	return out
}
