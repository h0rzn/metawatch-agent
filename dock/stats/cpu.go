package stats

import "github.com/docker/docker/api/types"

type CPU struct {
	UsagePerc float64 `json:"perc"`
	Online    float64 `json:"online"`
}

func NewCPU(preCPU, sysCPU types.CPUStats) *CPU {
	var cpuPerc = 0.0
	prevCPUU := preCPU.CPUUsage.TotalUsage
	prevSysU := preCPU.SystemUsage

	cpuDelta := float64(sysCPU.CPUUsage.TotalUsage) - float64(prevCPUU)
	systemDelta := float64(sysCPU.SystemUsage) - float64(prevSysU)

	online := float64(sysCPU.OnlineCPUs)

	if online == 0.0 {
		online = float64(len(sysCPU.CPUUsage.PercpuUsage))
	}
	if systemDelta > 0.0 {
		cpuPerc = (cpuDelta / systemDelta) * online * 100.0
	}

	return &CPU{
		UsagePerc: float64(cpuPerc),
		Online:    float64(online),
	}
}
