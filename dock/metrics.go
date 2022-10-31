package dock

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/h0rzn/monitoring_agent/dock/metrics"
)

type MetricsStreamer struct {
	mutex     *sync.Mutex
	Src       Source
	Consumers map[*Consumer]bool
	Reg       chan *Consumer
	Ureg      chan *Consumer
	Dis       chan *metrics.Set
}

type Consumer struct {
	In     chan *metrics.Set
	Single bool
}

func NewConsumer(single bool) *Consumer {
	return &Consumer{
		In:     make(chan *metrics.Set),
		Single: single,
	}
}

type Source struct {
	R    io.Reader
	Dec  *json.Decoder
	Done chan bool
}

func NewMetricsStreamer() *MetricsStreamer {
	return &MetricsStreamer{
		mutex:     &sync.Mutex{},
		Consumers: make(map[*Consumer]bool),
		Reg:       make(chan *Consumer),
		Ureg:      make(chan *Consumer),
		Dis:       make(chan *metrics.Set),
	}
}

func (s *MetricsStreamer) Source(r io.Reader, d chan bool) {
	src := Source{R: r, Dec: json.NewDecoder(r), Done: d}
	s.Src = src

}

func (s *MetricsStreamer) register(c *Consumer) {
	s.mutex.Lock()
	s.Consumers[c] = true
	s.mutex.Unlock()
}

func (s *MetricsStreamer) unregister(c *Consumer) {
	s.mutex.Lock()
	close(c.In)
	delete(s.Consumers, c)
	s.mutex.Unlock()
}

func (s *MetricsStreamer) Produce(dec *json.Decoder, die chan bool) <-chan types.StatsJSON {
	out := make(chan types.StatsJSON)
	go func() {
		for {
			select {
			case <-die:
				close(out)
				return
			default:
			}
			var stat types.StatsJSON
			err := dec.Decode(&stat)
			if err != nil {
				panic(err)
			}
			out <- stat
		}
	}()
	return out
}

func (s *MetricsStreamer) Process(in <-chan types.StatsJSON, die chan bool) <-chan *metrics.Set {
	out := make(chan *metrics.Set)
	go func() {

		for statsJson := range in {
			select {
			case <-die:
				close(out)
				return
			default:
			}
			set := metrics.NewSetWithJSON(statsJson)
			out <- &set
		}
		close(out)
	}()
	return out
}

// add done channel and close channels
func (s *MetricsStreamer) Run() {
	die := make(chan bool)
	prod := s.Produce(s.Src.Dec, die)
	proc := s.Process(prod, die)

	go func() {
		for {
			select {
			case c := <-s.Reg:
				fmt.Println("registering consumer")
				s.register(c)
			case c := <-s.Ureg:
				fmt.Println("unregisterung consumer")
				s.unregister(c)
			case data := <-s.Dis:
				for c := range s.Consumers {
					c.In <- data
					if c.Single { // unregister single dataset consumer after sending dataset
						fmt.Println("unregistering consumer after single dataset has been sent")
						s.unregister(c)
					}
				}
			case <-s.Src.Done:
				die <- true
			}
		}
	}()

	for set := range proc {
		s.Dis <- set
	}
}
