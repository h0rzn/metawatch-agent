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

func Average(metrics []Set) Set {
	var result Set
	c := len(metrics)
	cfloat := float64(c)

	for _, set := range metrics {
		result.CPU.UsagePerc += set.CPU.UsagePerc
		result.CPU.Online += set.CPU.Online

		result.Mem.Usage += set.Mem.Usage
		result.Mem.UsagePerc += set.Mem.UsagePerc
		result.Mem.Available += set.Mem.Available

		result.Disk.Read += set.Disk.Read
		result.Disk.Write += set.Disk.Write

		result.Net.In += set.Net.In
		result.Net.Out += set.Net.Out
	}

	result.When = metrics[len(metrics)-1].When

	result.CPU.UsagePerc = result.CPU.UsagePerc / cfloat
	result.CPU.Online = result.CPU.Online / cfloat

	result.Mem.Usage = result.Mem.Usage / cfloat
	result.Mem.UsagePerc = result.Mem.UsagePerc / cfloat
	result.Mem.Available = result.Mem.Available / cfloat

	result.Disk.Read = result.Disk.Read / cfloat
	result.Disk.Read = result.Disk.Read / cfloat

	result.Net.In = result.Net.In / cfloat
	result.Net.Out = result.Net.Out / cfloat

	return result
}

func Chunk(metrics []Set, chunkSize int) [][]Set {
	var chunks [][]Set
	for i := 0; i < len(metrics); i += chunkSize {
		end := i + chunkSize

		if end > len(metrics) {
			end = len(metrics)
		}

		chunks = append(chunks, metrics[i:end])
	}
	return chunks
}
