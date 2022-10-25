package stats

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

func ProduceStats(dec *json.Decoder) <-chan types.StatsJSON {
	out := make(chan types.StatsJSON)
	go func() {
		for {
			var stat types.StatsJSON
			_ = dec.Decode(&stat)
			out <- stat
		}
	}()
	return out
}

func GetFromStream(in <-chan types.StatsJSON) <-chan Set {
	out := make(chan Set)
	go func() {
		for stat := range in {
			set := NewSetWithJSON(stat)
			out <- set
		}
	}()
	return out
}
