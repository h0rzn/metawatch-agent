package stream

import "github.com/sirupsen/logrus"

type Receiver struct {
	In    chan Set
	Leave chan *Receiver
}

func NewReceiver(leaver chan *Receiver) *Receiver {
	return &Receiver{
		In:    make(chan Set),
		Leave: leaver,
	}
}

func (recv *Receiver) Quit() {
	logrus.Debugln("- RECEIVER - quit")
	close(recv.In)
	recv.Leave <- recv
}
