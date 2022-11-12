package stream

import "io"

type Streamer interface {
	Subscribe() Subscriber
	Unsubscribe(sub Subscriber)
	Run()
	Quit()
}

type Director interface {
	Source() (io.Reader, error)
	Subscribe() (Subscriber, error)
	Stream(chan interface{}) chan *Set
}
