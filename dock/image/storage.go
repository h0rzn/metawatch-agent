package image

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
)

type Storage struct {
	mutex  sync.Mutex
	c      *client.Client
	Images map[*Image]bool
	Feed   chan interface{}
}

func NewStorage(c *client.Client) *Storage {
	return &Storage{
		mutex:  sync.Mutex{},
		c:      c,
		Images: map[*Image]bool{},
		Feed:   make(chan interface{}),
	}
}

func (s *Storage) Init() error {
	ctx := context.Background()
	raws, err := s.c.ImageList(
		ctx,
		types.ImageListOptions{
			Filters: filters.Args{},
		})
	if err != nil {
		return err
	}
	logrus.Infof("- STORAGE - discovered %d image(s)\n", len(raws))

	for idx := range raws {
		s.AddRaw(raws[idx])
	}

	return nil
}

func (s *Storage) AddRaw(raw types.ImageSummary) error {
	s.mutex.Lock()
	img := NewImage(raw)
	s.Images[img] = true
	s.mutex.Unlock()
	return nil
}

func (s *Storage) Add(id string) error {
	s.mutex.Lock()
	if _, exists := s.Image(id); exists {
		return nil
	}
	ctx := context.Background()
	idFilter := filters.NewArgs()
	idFilter.Add("id", id)
	images, err := s.c.ImageList(
		ctx,
		types.ImageListOptions{
			Filters: idFilter,
		})
	if err != nil {
		return err
	}
	raw := images[0]
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
	if img, exists := s.Image(id); exists {
		delete(s.Images, img)
		logrus.Infof("- STORAGE - image removed: %d left\n", len(s.Images))
	} else {
		logrus.Warningln("- STORAGE - tried to remove unkown image")
	}
	s.mutex.Unlock()

	return nil
}

func (s *Storage) ByID(id string) (*Image, bool) {
	for img := range s.Images {
		if img.ID == id {
			return img, true
		}
	}
	return &Image{}, false
}

func (s *Storage) Image(id string) (*Image, bool) {
	for img := range s.Images {
		if img.ID == id {
			return img, true
		}
	}
	return &Image{}, false
}

func (s *Storage) Items() (images []*Image) {
	s.mutex.Lock()
	for img := range s.Images {
		images = append(images, img)
	}
	s.mutex.Unlock()
	return
}

func (s *Storage) MarshalJSON() ([]byte, error) {
	var images []*Image
	for img := range s.Images {
		images = append(images, img)
	}

	return json.Marshal(images)
}
