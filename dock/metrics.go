package dock

import (
	"encoding/json"
	"io"
	"sync"

	"github.com/docker/docker/api/types"
)

type MetricsStreamer struct {
	mutex     sync.Mutex
	Src       Source
	Consumers map[*Consumer]bool
	Reg       chan *Consumer
	Ureg      chan *Consumer
	Dis       chan *DataSet
}

type DataSet struct{}

type Consumer struct {
	In     chan *DataSet
	Single bool
}

func NewConsumer(single bool) *Consumer {
	return &Consumer{
		In:     make(chan *DataSet),
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
		mutex:     sync.Mutex{},
		Consumers: make(map[*Consumer]bool),
		Reg:       make(chan *Consumer),
		Ureg:      make(chan *Consumer),
		Dis:       make(chan *DataSet),
	}
}

func (s *MetricsStreamer) Source(r io.Reader, d chan bool) {
	src := Source{R: r, Dec: json.NewDecoder(r), Done: d}
	s.Src = src
}

func (s *MetricsStreamer) Register(c *Consumer) {
	s.mutex.Lock()
	s.Consumers[c] = true
	s.mutex.Unlock()
}

func (s *MetricsStreamer) Unregister(c *Consumer) {
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
			_ = dec.Decode(&stat)
			out <- stat
		}
	}()
	return out
}

func (s *MetricsStreamer) Process(in <-chan types.StatsJSON, die chan bool) <-chan *DataSet {
	out := make(chan *DataSet)
	go func() {
		select {
		case <-die:
			close(out)
			return
		default:
		}
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
func (s *MetricsStreamer) Run() {
	die := make(chan bool)
	prod := s.Produce(s.Src.Dec, die)
	proc := s.Process(prod, die)

	go func() {
		for {
			select {
			case c := <-s.Reg:
				s.Register(c)
			case c := <-s.Ureg:
				s.Unregister(c)
			case data := <-s.Dis:
				for c := range s.Consumers {
					c.In <- data
					if c.Single { // unregister single dataset consumer after sending dataset
						s.Unregister(c)
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
