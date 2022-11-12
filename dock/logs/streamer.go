package logs

import (
	"encoding/binary"
	"fmt"
	"io"
	"strings"
	"sync"
)

type Streamer struct {
	mutex *sync.Mutex
	r     io.Reader
	Subs  map[*Sub]bool
	Done  chan bool
}

func NewStreamer(r io.Reader) *Streamer {
	return &Streamer{
		mutex: &sync.Mutex{},
		r:     r,
		Subs:  make(map[*Sub]bool),
		Done:  make(chan bool),
	}
}

func (s *Streamer) quit() {
	s.mutex.Lock()
	for sub := range s.Subs {
		close(sub.in)
	}
	s.mutex.Unlock()
}

func (s *Streamer) Sub() *Sub {
	sub := NewSub()
	fmt.Println("[SUB]")
	s.mutex.Lock()
	s.Subs[sub] = true
	s.mutex.Unlock()
	return sub
}

func (s *Streamer) USub(sub *Sub) {
	fmt.Println("[USUB]")
	s.mutex.Lock()
	close(sub.in)
	delete(s.Subs, sub)
	if len(s.Subs) == 0 {
		fmt.Println("no subs left: closing streamer")
		s.quit()
	}
	s.mutex.Unlock()
}

func (s *Streamer) parse(d chan bool) (<-chan *Entry, <-chan error) {
	out := make(chan *Entry)
	errChan := make(chan error)

	go func() {
		for {
			hdr := make([]byte, 8)

			select {
			case <-d:
				close(out)
				close(errChan)
			default:
			}

			_, err := s.r.Read(hdr)
			if err != nil {
				if err == io.EOF {
					fmt.Println("eof")
				} else {
					errChan <- err
				}
			}

			sizes := binary.BigEndian.Uint32(hdr[4:])
			content := make([]byte, sizes)
			_, err = s.r.Read(content)
			_ = err
			if err != nil && err != io.EOF {
				errChan <- err
			}
			time, data, found := strings.Cut(string(content), " ")
			if found {
				entry := NewEntry(time, data, hdr[0])
				out <- entry
			}
		}
	}()
	return out, errChan
}

func (s *Streamer) Run() {
	done := make(chan bool)
	logs, errChan := s.parse(done)
	fmt.Println("parser running")

	for {
		select {
		case entry := <-logs:
			if len(s.Subs) > 0 {
				for sub := range s.Subs {
					sub.in <- entry
				}
			} else {
				s.quit()
			}
		case err := <-errChan:
			fmt.Println("parse err", err.Error())
		}
	}

}
