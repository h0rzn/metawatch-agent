package stream

type Streamer interface {
	Subscribe() *Subscriber
	Unsubscribe(sub Subscriber)
	Run()
	Quit()
}
