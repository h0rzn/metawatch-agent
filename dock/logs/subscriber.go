package logs

import (
	"fmt"
	"sync"
)

type Sub struct {
	mutex *sync.Mutex
	in    chan *Entry
}

func NewSub() *Sub {
	return &Sub{
		mutex: &sync.Mutex{},
		in:    make(chan *Entry),
	}
}

func (sub *Sub) Handle(out chan *Entry) {
	for entry := range sub.in {
		fmt.Println("out...")
		out <- entry
	}
}
