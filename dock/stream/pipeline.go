package stream

type Pipeline interface {
	Stop()
	Out() chan Set
}
