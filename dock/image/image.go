package image

import "github.com/docker/docker/api/types"

type Image struct {
	ID      string `json:"id"`
	Tag     string `json:"tag"`
	Size    int64  `json:"size"`
	Created int64  `json:"created"`
}

func NewImage(raw types.ImageSummary) *Image {
	return &Image{
		ID:      raw.ID,
		Tag:     raw.RepoTags[0],
		Size:    raw.Size,
		Created: raw.Created,
	}
}
