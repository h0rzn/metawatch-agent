package stream

import "github.com/sirupsen/logrus"

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
		In:      make(chan Set),
		Leave:   leave,
		Closing: make(chan struct{}),
	}
}

func (recv *Receiver) Close(byStreamer bool) {
	logrus.Debugln("- RECEIVER - close")
	if byStreamer {
		logrus.Debugln("- RECEIVER - send closing sig to consumers")
		recv.Closing <- struct{}{}
	} else {
		recv.Leave <- recv
	}
}
