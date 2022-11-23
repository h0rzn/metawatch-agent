package controller

import (
	"context"
	"fmt"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/h0rzn/monitoring_agent/dock/container"
)

// Storage stores container instances and manages changes
type Storage struct {
	mutex      sync.Mutex
	Containers []*container.Container
	c          *client.Client
}

func NewStorage(c *client.Client) *Storage {
	return &Storage{
		mutex:      sync.Mutex{},
		Containers: []*container.Container{},
		c:          c,
	}
}

func (s *Storage) Discover() ([]types.Container, error) {
	fmt.Println("discovering...")
	ctx := context.Background()
	containers, err := s.c.ContainerList(
		ctx,
		types.ContainerListOptions{
			Filters: filters.Args{},
		})
	fmt.Printf("discovered %d containers\n", len(containers))
	return containers, err
}

func (s *Storage) AddAll(containers []types.Container) error {
	var wg sync.WaitGroup
	var errors []error
	mutex := &sync.Mutex{}
	fmt.Println("adding containers")

	for _, raw := range containers {

		wg.Add(1)
		curRaw := raw
		go func() {
			defer wg.Done()
			err := s.Add(curRaw)
			fmt.Println("container add err", err)
			mutex.Lock()
			if err != nil {
				errors = append(errors, err)
			}
			mutex.Unlock()
		}()
	}

	wg.Wait()
	fmt.Printf("%d errors while  adding %d containers\n", len(errors), len(containers))
	return nil
}

func (s *Storage) Add(raw types.Container) error {
	fmt.Println("adding container", raw.Names)
	container := container.NewContainer(raw, s.c)
	err := container.Start()
	if err != nil {
		return err
	}
	fmt.Println("container successfully started")
	s.mutex.Lock()
	s.Containers = append(s.Containers, container)
	fmt.Println("created and added container")
	s.mutex.Unlock()
	return nil
}

func (s *Storage) Container(id string) *container.Container {
	for _, container := range s.Containers {
		if container.ID == id {
			return container
		}
	}
	return nil
}

func (s *Storage) JSONSkel() []*container.ContainerJSON {
	var skels []*container.ContainerJSON

	for _, container := range s.Containers {
		skels = append(skels, container.JSONSkel())
	}
	return skels
}
