package stream

type Subscriber interface {
	Handle(out chan *Set)
}

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
