package metrics

import (
	"encoding/json"
	"io"
	"time"

	"github.com/docker/docker/api/types"
)

type Set struct {
	When time.Time `json:"time"`
	CPU  CPU       `json:"cpu"`
	Mem  Memory    `json:"memory"`
	Disk Disk      `json:"disk"`
	Net  Net       `json:"net"`
}

func NewSet(r io.Reader) Set {
	var jsonStats types.StatsJSON
	dec := json.NewDecoder(r)

	_ = dec.Decode(&jsonStats)
	return NewSetWithJSON(jsonStats)
}

func NewSetWithJSON(stats types.StatsJSON) Set {
	cpu := NewCPU(stats.PreCPUStats, stats.CPUStats)
	mem := NewMem(stats.MemoryStats)
	disk := NewDisk(stats.BlkioStats)
	net := NewNet(stats.Networks)

	return Set{
		When: time.Time{},
		CPU:  *cpu,
		Mem:  *mem,
		Disk: *disk,
		Net:  *net,
	}
}
