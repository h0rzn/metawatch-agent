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
	tag := "<no_tag>"
	if len(raw.RepoTags) > 0 {
		tag = raw.RepoTags[0]
	}

	return &Image{
		ID:         raw.ID,
		Tag:        tag,
		Size:       raw.Size,
		Created:    stamp,
		Containers: raw.Containers,
	}
}
