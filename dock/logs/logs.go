package logs

import "sync"

// add logic to "ignore" if no consumers are registered
// as the logs are not supposed to be stored
// maybe request logs from docker engine on demand?
type LogStreamer struct {
	mutex     *sync.Mutex
	Consumers map[*Consumer]bool
	Reg       chan *Consumer
	Ureg      chan *Consumer
	Dis       chan *Entry
}

type Consumer struct {
}

func NewLogStreamer() *LogStreamer {
	return &LogStreamer{
		mutex:     &sync.Mutex{},
		Consumers: make(map[*Consumer]bool),
		Reg:       make(chan *Consumer),
		Ureg:      make(chan *Consumer),
		Dis:       make(chan *Entry),
	}
}

func (s *LogStreamer) Run() {
	// set pipeline

	go func() {
		select {
		case c := <-s.Reg:
			s.mutex.Lock()
			s.Consumers[c] = true
			s.mutex.Unlock()
		case c := <-s.Ureg:
			s.mutex.Lock()
			// close input channel of consumer
			delete(s.Consumers, c)
			s.mutex.Unlock()
		case entry := <-s.Dis:
			for consumer := range s.Consumers {
				_, _ = consumer, entry
				// send data to consumer
			}
			// add die/done channel handling
		}
	}()

	// distribute entries

}
