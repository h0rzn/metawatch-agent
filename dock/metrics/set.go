package metrics

import (
	"encoding/json"
	"io"
	"time"

	"github.com/docker/docker/api/types"
)

type Set struct {
	When string `json:"when"`
	CPU  CPU    `json:"cpu"`
	Mem  Memory `json:"memory"`
	Disk Disk   `json:"disk"`
	Net  Net    `json:"net"`
}

func NewSet(r io.Reader) Set {
	var jsonStats types.StatsJSON
	dec := json.NewDecoder(r)

	_ = dec.Decode(&jsonStats)
	return NewSetWithJSON(jsonStats)
}

func NewSetWithJSON(stats types.StatsJSON) Set {
	return Set{
		When: stats.Read.Format(time.RFC3339Nano),
		CPU:  *NewCPU(stats.PreCPUStats, stats.CPUStats),
		Mem:  *NewMem(stats.MemoryStats),
		Disk: *NewDisk(stats.BlkioStats),
		Net:  *NewNet(stats.Networks),
	}
}
