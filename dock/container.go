package dock

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/h0rzn/monitoring_agent/dock/stats"
)

type Container struct {
	ID          string
	Names       []string
	Image       string
	State       ContainerState
	StatsReader io.Reader        // passed down from controller to access stats stream
	StatsWriter chan interface{} // send processed stats data back
}

type ContainerState struct {
	Status  string // "running", "stopped", ...
	Started string
}

func (c *Container) ReadStats(r io.Reader) {
	dec := json.NewDecoder(r)

	for {
		var statsJson *types.StatsJSON

		err := dec.Decode(&statsJson)
		if err != nil {
			if err == io.EOF {
				break
			} else {
				panic(err)
			}
		}

		//cpu := stats.NewCPU(statsJson.PreCPUStats, statsJson.CPUStats)
		//mem := stats.NewMem(statsJson.MemoryStats)
		//fmt.Printf("MEM: %f (%f/%f) CPU: %f\n", mem.UsagePerc, mem.Usage, mem.Available, cpu.UsagePerc)
		net := stats.NewNet(statsJson.Networks)
		fmt.Println(net.In, net.Out)

	}
}
