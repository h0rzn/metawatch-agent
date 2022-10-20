package dock

import (
	"github.com/h0rzn/monitoring_agent/dock/stats"
)

type StatsSet struct {
	CPU    stats.CPU `json:"cpu_stats"`
	PreCPU stats.CPU `json:"precpu_stats"`
}
