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
	"github.com/h0rzn/monitoring_agent/dock/image"
	"github.com/sirupsen/logrus"
)

const (
	bulkWriteN      int = 5
	containerExists     = iota
	containerStartErr
	containerAdded
)

// Storage stores container instances and manages changes
// container Images are modified and passed through from docker engine api
type Storage struct {
	mutex          sync.RWMutex
	ContainerStore map[*container.Container]bool
	c              *client.Client
	DB             *db.DB
	Events         *Events
	FeedIn         chan interface{}
}

func NewStorage(c *client.Client) *Storage {
	strg := &Storage{
		mutex:          sync.RWMutex{},
		ContainerStore: make(map[*container.Container]bool),
		c:              c,
		FeedIn:         make(chan interface{}),
		DB:             &db.DB{},
	}
	strg.Events = NewEvents(c, strg)
	go strg.Events.Run()
	return strg
}

func (s *Storage) Init() error {
	raw, err := s.Discover(filters.Args{})
	if err != nil {
		return err
	}

	err = s.Add(raw...)
	if err != nil {
		return err
	}

	err = s.DB.Init()
	if err != nil {
		return err
	}

	go s.Feed()

	var images []*image.Image
	ctx := context.Background()
	imgs, err := s.c.ImageList(ctx, types.ImageListOptions{})
	if err != nil {
		fmt.Println(err)
	}
	for _, img := range imgs {
		imgInsp, _, err := s.c.ImageInspectWithRaw(ctx, img.ID)
		if err != nil {
			fmt.Println(err)
		}

		cur := &image.Image{Tag: imgInsp.RepoTags[0]}
		images = append(images, cur)
	}
	for _, img := range images {
		fmt.Println(img)
	}

	return nil
}

func (s *Storage) Images() (images []*image.Image, err error) {
	ctx := context.Background()
	imgsRaw, err := s.c.ImageList(ctx, types.ImageListOptions{})
	if err != nil {
		return
	}

	for _, raw := range imgsRaw {
		images = append(images, image.NewImage(raw))
	}
	return
}

func (s *Storage) Containers() (containers []*container.Container, err error) {
	s.mutex.Lock()
	for container := range s.ContainerStore {
		containers = append(containers, container)
	}
	s.mutex.Unlock()
	return
}

func (s *Storage) Discover(filters filters.Args) ([]types.Container, error) {
	ctx := context.Background()
	containers, err := s.c.ContainerList(
		ctx,
		types.ContainerListOptions{
			Filters: filters,
		})
	logrus.Infof("- STORAGE - discovered %d container(s)\n", len(containers))
	return containers, err
}

func (s *Storage) Add(raw ...types.Container) error {
	var added int
	for _, rawCont := range raw {
		cont := container.NewContainer(rawCont, s.c, s.FeedIn)
		err := cont.Start()
		if err != nil {
			logrus.Errorf("- STORAGE - container failed to start (ignore): %s", err)
			continue
		}
		s.ContainerStore[cont] = true
		added = added + 1
	}
	logrus.Infof("- STORAGE - added %d container(s)\n", added)
	return nil
}

// Push adds a container when its existance in storage is unkown
// used by events
func (s *Storage) Push(id string) int {
	if _, exists := s.Container(id); exists {
		return containerExists
	}

	f := filters.NewArgs()
	f.Add("id", id)
	found, err := s.Discover(f)
	if err != nil || len(found) != 1 {
		return containerStartErr
	}

	err = s.Add(found...)
	if err != nil {
		return containerStartErr
	}

	return containerAdded
}

func (s *Storage) Remove(cid string) error {
	logrus.Debugf("- STORAGE - attempting to remove container %s\n", cid)
	s.mutex.Lock()
	if container, exists := s.Container(cid); exists {
		err := container.Stop()
		if err != nil {
			return err
		}

		delete(s.ContainerStore, container)
		logrus.Infof("- STORAGE - container removed: %d left\n", len(s.ContainerStore))
	} else {
		logrus.Warningln("- STORAGE - tried to remove non-existent container")
	}
	s.mutex.Unlock()

	return nil
}

func (s *Storage) Container(id string) (*container.Container, bool) {
	for container := range s.ContainerStore {
		if container.ID == id {
			return container, true
		}
	}
	return &container.Container{}, false
}

// func (s *Storage) JSONSkel() []*container.ContainerJSON {
// 	var skels []*container.ContainerJSON

// 	for container := range s.Containers {
// 		skels = append(skels, container.JSONSkel())
// 	}
// 	return skels

// }

func (s *Storage) Feed() {
	data := []interface{}{}
	for set := range s.FeedIn {
		data = append(data, set)
		if len(data) >= bulkWriteN {
			s.DB.Client.BulkWrite(data)
			data = nil
		}
	}
}
