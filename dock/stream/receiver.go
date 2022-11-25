package stream

import "fmt"

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
	fmt.Println("[RECEIVER] quit")
	close(recv.In)
	recv.Leave <- recv
}
