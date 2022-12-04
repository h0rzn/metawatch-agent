package stream

import "github.com/sirupsen/logrus"

type Receiver struct {
	// interval streamer
	Interv bool
	In     chan Set
	Leave  chan *Receiver
}

func NewReceiver(interv bool, leaver chan *Receiver) *Receiver {
	return &Receiver{
		Interv: interv,
		In:     make(chan Set),
		Leave:  leaver,
	}
}

func (recv *Receiver) Quit() {
	logrus.Debugln("- RECEIVER - quit")
	//close(recv.In)
	recv.Leave <- recv
}
