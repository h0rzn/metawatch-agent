package stream

import (
	"io"
)

type Director interface {
	Source() (io.Reader, error)
	Subscribe() (Subscriber, error)
	Stream(chan struct{}) chan *Set
}

type Streamer interface {
	Subscribe() Subscriber
	Unsubscribe(sub Subscriber)
	Run()
	Quit()
}
