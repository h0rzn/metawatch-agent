package metrics

import "github.com/docker/docker/api/types"

type Memory struct {
	UsagePerc float64 `json:"perc"`
	Usage     float64 `json:"usage_bytes"`
	Available float64 `json:"available_bytes"`
}

func NewMem(mem types.MemoryStats) *Memory {
	var memU = 0.0
	var memP = 0.0
	var limit = float64(mem.Limit)

	// add support for cgroup v1
	// cgroup v2
	if v := mem.Stats["inactive_file"]; v < mem.Usage {
		memU = float64(mem.Usage - v)
	}

	// in percent
	if limit != 0 { // memLimit, memU
		memP = memU / float64(limit) * 100
	}

	return &Memory{
		UsagePerc: memP,
		Usage:     memU,
		Available: limit,
	}
}
