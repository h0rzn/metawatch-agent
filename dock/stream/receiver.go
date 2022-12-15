package stream

import (
	"github.com/sirupsen/logrus"
)

type Receiver struct {
	// interval streamer
	Interv  bool
	In      chan Set
	Leave   chan *Receiver
	Closing chan struct{}
}

func NewReceiver(interv bool, leave chan *Receiver) *Receiver {
	return &Receiver{
		Interv:  interv,
		In:      make(chan Set, 1),
		Leave:   leave,
		Closing: make(chan struct{}, 1),
	}
}

// Quit handles intrinsic motivated leave
func (recv *Receiver) Quit() {
	logrus.Debugln("- RECEIVER - quit")
	recv.Leave <- recv
}

func (recv *Receiver) Close() {
	logrus.Debugln("- RECEIVER - close")
	recv.Closing <- struct{}{}
}
