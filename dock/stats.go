package dock

import (
	"encoding/json"
	"io"
	"sync"

	"github.com/docker/docker/api/types"
)

type StatStreamer struct {
	mutex     sync.Mutex
	Consumers map[*Consumer]bool
	Reg       chan *Consumer
	Ureg      chan *Consumer
	Dis       chan *DataSet
	Shut      chan int
}

func (s *StatStreamer) SetReader(r io.Reader) {}

func (s *StatStreamer) Register(c *Consumer) {
	s.mutex.Lock()
	s.Consumers[c] = true
	s.mutex.Unlock()
}

func (s *StatStreamer) Unregister(c *Consumer) {
	s.mutex.Lock()
	close(c.In)
	delete(s.Consumers, c)
	s.mutex.Unlock()
}

func (s *StatStreamer) Produce(dec *json.Decoder) <-chan types.StatsJSON {
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

func (s *StatStreamer) Shutdown() {}

func (s *StatStreamer) Process(in <-chan types.StatsJSON) <-chan *DataSet {
	out := make(chan *DataSet)
	go func() {
		for statsJson := range in {
			_ = statsJson
			set := DataSet{}
			out <- &set
		}
		close(out)
	}()
	return out
}

// add done channel and close channels
func (s *StatStreamer) Run() {
	for {
		select {
		case c := <-s.Reg:
			s.Register(c)
		case c := <-s.Ureg:
			s.Unregister(c)
		case data := <-s.Dis:
			for c := range s.Consumers {
				c.In <- data
			}
		}
	}
}
