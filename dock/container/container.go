package container

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/docker/docker/client"
	"github.com/h0rzn/monitoring_agent/dock/controller/db"
	"github.com/h0rzn/monitoring_agent/dock/logs"
	"github.com/h0rzn/monitoring_agent/dock/metrics"
	"github.com/sirupsen/logrus"
)

const feederInterv = 5 * time.Second

type Container struct {
	ID       string              `json:"id"`
	Names    string              `json:"name"`
	Image    string              `json:"image"`
	State    State               `json:"state"`
	Networks []*Network          `json:"networks"`
	Volumes  []*Volume           `json:"volume"`
	Ports    map[string][]string `json:"ports"`
	Streams  Streams             `json:"-"`
	c        *client.Client      `json:"-"`
}

type State struct {
	Status        string `json:"status"`
	Started       string `json:"since"`
	RestartPolicy string `json:"restart_policy"`
}

type Network struct {
	Name    string   `json:"name"`
	ID      string   `json:"id"`
	Aliases []string `json:"aliases"`
	IPAddr  string   `json:"ip"`
}

type Volume struct {
	Path string `json:"path"`
}

type Streams struct {
	Logs       *logs.Logs
	Metrics    *metrics.Metrics
	FeederDone chan struct{}
	FeedIn     chan interface{}
}

// Feed collects data from streamer and feeds them to tb
func (s *Streams) Feed(cid string) chan interface{} {
	out := make(chan interface{})

	logrus.Debugln("- CONTAINER - starting feed")

	metricsRcv, err := s.Metrics.Get(true)
	if err != nil {
		close(out)
		return out
	}

	go func() {
		defer close(out)

		feederTicker := time.NewTicker(feederInterv)
		for {
			select {
			case <-s.FeederDone:
				return
			case <-metricsRcv.Closing:
				return
			case <-feederTicker.C:
				set, ok := <-metricsRcv.In
				if !ok {
					fmt.Println("feeder: rcv in is closed")
					return
				}
				if metricsSet, ok := set.Data.(metrics.Set); ok {
					// s.FeedIn <- db.NewMetricsMod(cid, metricsSet.When, metricsSet)
					out <- db.NewMetricsMod(cid, metricsSet.When, metricsSet)
					logrus.Debugf("- Container - -> sending for cid: %s\n", cid)
				} else {
					logrus.Info("- LINK - failed to parse incomming interface to metrics.Set")
				}
			}
		}
	}()
	return out
}

func (s *Streams) Stop() error {
	logrus.Debugln("- CONTAINER - stop: stopping streams and feeder")
	s.FeederDone <- struct{}{}
	fmt.Println("feeder done sent")

	err := s.Metrics.Stop()
	if err != nil {
		return err
	}
	logrus.Debugln("- CONTAINER - metrics stopped")

	err = s.Logs.Stop()
	if err != nil {
		return err
	}
	logrus.Debugln("- CONTAINER - logs stopped")

	return nil
}

func NewContainer(c *client.Client, cid string, feedIn chan interface{}) *Container {
	return &Container{
		ID:       cid,
		Networks: make([]*Network, 0),
		Volumes:  make([]*Volume, 0),
		Ports:    make(map[string][]string),
		Streams: Streams{
			FeederDone: make(chan struct{}),
			FeedIn:     feedIn,
		},
		c: c,
	}
}

func (cont *Container) prepare() <-chan error {
	out := make(chan error, 1)

	// inspect container for further info
	ctx := context.Background()
	json, err := cont.c.ContainerInspect(ctx, cont.ID)
	if err != nil {
		fmt.Println("inspect err", err)
		out <- err
		return out
	}
	base := json.ContainerJSONBase

	cont.Names = base.Name
	cont.Image = base.Image

	// ports
	ports := json.NetworkSettings.Ports
	for port, binds := range ports {
		p := fmt.Sprintf("%s/%s", port.Port(), port.Proto())
		for _, b := range binds {
			bFmt := fmt.Sprintf("%s:%s", b.HostIP, b.HostPort)
			cont.Ports[p] = append(cont.Ports[p], bFmt)
		}
	}

	// set state
	cont.State = State{
		Status:        base.State.Status,
		Started:       base.State.StartedAt,
		RestartPolicy: base.HostConfig.RestartPolicy.Name,
	}

	// networks
	for name, eps := range json.NetworkSettings.Networks {
		net := &Network{
			Name:    name,
			Aliases: eps.Aliases,
			IPAddr:  eps.IPAddress,
		}
		cont.Networks = append(cont.Networks, net)
	}

	// volumes
	for path := range json.Config.Volumes {
		cont.Volumes = append(cont.Volumes, &Volume{Path: path})
	}

	// streams
	cont.Streams.Metrics = metrics.NewMetrics(cont.c, base.ID)
	cont.Streams.Logs = logs.NewLogs(cont.c, base.ID)

	out <- err
	return out
}

func (cont *Container) Start() error {
	logrus.Infoln("- CONTAINER - preparing...")
	select {
	case err := <-cont.prepare():
		go func() {
			for set := range cont.Streams.Feed(cont.ID) {
				cont.Streams.FeedIn <- set
			}
		}()
		return err
	case <-time.After(8 * time.Second):
		return errors.New("container start timed out")
	}
}

func (cont *Container) Stop() error {
	logrus.Infoln("- CONTAINER - stop")
	return cont.Streams.Stop()
}

func (cont *Container) MarshalJSON() ([]byte, error) {
	type Alias Container
	var currentMetrics metrics.Set

	recv, err := cont.Streams.Metrics.Get(false)
	if err != nil {
		return []byte{}, err
	}
	for cur := range recv.In {
		currentMetrics = cur.Data.(metrics.Set)
		break
	}
	recv.Close()

	return json.Marshal(&struct {
		CurMetrics metrics.Set `json:"metrics"`
		*Alias
	}{
		CurMetrics: currentMetrics,
		Alias:      (*Alias)(cont),
	})
}
