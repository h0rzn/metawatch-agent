package stream

type Subscriber interface {
	Handle(out chan *Set)
	Put(*Set)
	Quit()
}

// TempSub temporarily subscribes to stream
type TempSub struct {
	In chan *Set
}

func NewTempSub() *TempSub {
	return &TempSub{
		In: make(chan *Set),
	}
}

func (sub *TempSub) Handle(out chan *Set) {
	for set := range sub.In {
		out <- set
	}
}

func (sub *TempSub) Put(set *Set) {
	sub.In <- set
}

func (sub *TempSub) Quit() {
	close(sub.In)
}
