package hub

import (
	"fmt"
	"sync"
	"time"

	"github.com/h0rzn/monitoring_agent/dock/container"
	"github.com/h0rzn/monitoring_agent/dock/metrics"
	"github.com/h0rzn/monitoring_agent/dock/stream"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type CombindedMetrics struct {
	ContainerStore *container.Storage
	mutex          *sync.Mutex
	Typ            string
	Subs           map[*Client]bool
	Timeout        *Timeout
	LveSig         chan Resource
	done           chan struct{}
}

func NewCombinedR(store *container.Storage, lveSig chan Resource) *CombindedMetrics {
	r := &CombindedMetrics{
		ContainerStore: store,
		mutex:          &sync.Mutex{},
		Typ:            "combined_metrics",
		Subs:           make(map[*Client]bool),
		LveSig:         lveSig,
		done:           make(chan struct{}),
	}
	r.Timeout = NewTimeout(r.Quit)
	return r
}

func (cm *CombindedMetrics) Type() string {
	return cm.Typ
}

func (cm *CombindedMetrics) CID() string {
	return "_all"
}

func (cm *CombindedMetrics) Add(c *Client) {
	if len(c.Sub) == 0 {
		cm.Timeout.Stop()
	}
	cm.mutex.Lock()
	cm.Subs[c] = true
	cm.mutex.Unlock()
}

func (cm *CombindedMetrics) Rm(c *Client) {
	cm.mutex.Lock()
	delete(cm.Subs, c)
	cm.mutex.Unlock()
	if len(cm.Subs) == 0 {
		cm.Timeout.Start()
	}
}

func (r *CombindedMetrics) Broadcast(set stream.Set) {
	frame := &Response{
		CID:     "_all",
		Type:    "combined_metrics",
		Message: set.Data,
	}
	r.mutex.Lock()
	for client := range r.Subs {
		client.In <- frame
	}
	r.mutex.Unlock()
}

func (cm *CombindedMetrics) procLatest(latest []metrics.Set) metrics.Set {
	var result metrics.Set
	result.When = primitive.NewDateTimeFromTime(time.Now())

	for _, set := range latest {
		// cpu
		result.CPU.UsagePerc += set.CPU.UsagePerc
		result.CPU.Online += set.CPU.Online
		// mem
		result.Mem.Usage += set.Mem.Usage
		result.Mem.UsagePerc += set.Mem.UsagePerc
		result.Mem.Available += set.Mem.Available
		// disk
		result.Disk.Read += set.Disk.Read
		result.Disk.Write += set.Disk.Write
		// net
		result.Net.In += set.Net.In
		result.Net.Out += set.Net.Out
	}
	return result
}

func (cm *CombindedMetrics) Latest() chan metrics.Set {
	out := make(chan metrics.Set)

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer close(out)

		comb := cm.procLatest(cm.ContainerStore.CollectLatest())
		out <- comb

		for {
			select {
			case <-cm.done:
				return
			case <-ticker.C:
				comb := cm.procLatest(cm.ContainerStore.CollectLatest())
				out <- comb
			}
		}
	}()
	return out
}

func (cm *CombindedMetrics) Run() error {
	fmt.Println("combined running")
	combined := cm.Latest()
	go func() {
		for set := range combined {
			set := *stream.NewSet("combined_metrics", set)
			cm.Broadcast(set)
		}
	}()
	return nil
}

func (cm *CombindedMetrics) Quit() {
	fmt.Println("comined metrics: quit")
	cm.mutex.Lock()
	cm.done <- struct{}{}
	for c := range cm.Subs {
		cm.Rm(c)
	}
	cm.LveSig <- cm
	cm.mutex.Unlock()
}
