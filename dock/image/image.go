package image

import (
	"time"

	"github.com/docker/docker/api/types"
)

type Image struct {
	ID         string `json:"id"`
	Tag        string `json:"tag"`
	Size       int64  `json:"size"`
	Created    string `json:"created"`
	Containers int64  `json:"containers"`
}

func NewImage(raw types.ImageSummary) *Image {
	unix := time.Unix(raw.Created, 0)
	stamp := unix.Format(time.RFC3339Nano)

	return &Image{
		ID:         raw.ID,
		Tag:        raw.RepoTags[0],
		Size:       raw.Size,
		Created:    stamp,
		Containers: raw.Containers,
	}
}
