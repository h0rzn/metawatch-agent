package metrics

import (
	"encoding/json"
	"io"

	"github.com/docker/docker/api/types"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Set struct {
	When primitive.DateTime `json:"when" bson:"-"`
	CPU  CPU                `json:"cpu" bson:"cpu,inline"`
	Mem  Memory             `json:"memory" bson:"mem,inline"`
	Disk Disk               `json:"disk" bson:"disk,inline"`
	Net  Net                `json:"net" bson:"net,inline"`
}

func NewSet(r io.Reader) Set {
	var jsonStats types.StatsJSON
	dec := json.NewDecoder(r)

	_ = dec.Decode(&jsonStats)
	return NewSetWithJSON(jsonStats)
}

func NewSetWithJSON(stats types.StatsJSON) Set {
	return Set{
		When: primitive.NewDateTimeFromTime(stats.Read), //stats.Read.Format(time.RFC3339Nano),
		CPU:  *NewCPU(stats.PreCPUStats, stats.CPUStats),
		Mem:  *NewMem(stats.MemoryStats),
		Disk: *NewDisk(stats.BlkioStats),
		Net:  *NewNet(stats.Networks),
	}
}
