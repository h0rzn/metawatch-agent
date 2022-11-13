package metrics

import (
	"encoding/json"
	"io"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/h0rzn/monitoring_agent/dock/stream"
)

type Streamer struct {
	mutex *sync.Mutex
	Src   Source
	Subs  map[stream.Subscriber]bool
}

type Source struct {
	R   io.Reader
	Dec *json.Decoder
}

func NewStreamer(r io.Reader) *Streamer {
	return &Streamer{
		mutex: &sync.Mutex{},
		Src: Source{
			R:   r,
			Dec: json.NewDecoder(r),
		},
		Subs: make(map[stream.Subscriber]bool),
	}
}

func (s *Streamer) Subscribe() stream.Subscriber {
	sub := stream.NewTempSub()
	s.mutex.Lock()
	s.Subs[sub] = true
	s.mutex.Unlock()
	return sub
}

func (s *Streamer) Unsubscribe(sub stream.Subscriber) {
	s.mutex.Lock()
	sub.Quit()
	delete(s.Subs, sub)
	s.mutex.Unlock()
}

// Produce decodes bytes to types.StatsJSON and returns a live channel with its data
func (s *Streamer) Produce(dec *json.Decoder, die chan bool) <-chan types.StatsJSON {
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

// Process receives the types.StatsJSON channel values and returns channel with live parsed metric sets
func (s *Streamer) Process(in <-chan types.StatsJSON, die chan bool) <-chan *Set {
	out := make(chan *Set)
	go func() {

		for statsJson := range in {
			select {
			case <-die:
				close(out)
				return
			default:
			}
			set := NewSetWithJSON(statsJson)
			out <- &set
		}
		close(out)
	}()
	return out
}

// add done channel and close channels
func (s *Streamer) Run() {
	done := make(chan bool)
	prod := s.Produce(s.Src.Dec, done)
	proc := s.Process(prod, done)

	for metric := range proc {
		for sub := range s.Subs {
			sub.Put(stream.NewSet("metric_set", metric))
		}
	}

}

func (s *Streamer) Quit() {
	s.mutex.Lock()
	for sub := range s.Subs {
		sub.Quit()
	}
	s.mutex.Unlock()
}
