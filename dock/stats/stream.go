package stats

import (
	"encoding/json"
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

func Produce(dec *json.Decoder) <-chan types.StatsJSON {
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

func Process(in <-chan types.StatsJSON) <-chan Set {
	out := make(chan Set)
	go func() {
		for stat := range in {
			cpu := NewCPU(stat.PreCPUStats, stat.CPUStats)
			mem := NewMem(stat.MemoryStats)
			disk := NewDisk(stat.BlkioStats)
			net := NewNet(stat.Networks)

			set := Set{
				When: time.Time{},
				CPU:  *cpu,
				Mem:  *mem,
				Disk: *disk,
				Net:  *net,
			}
			out <- set
		}
	}()
	return out
}
