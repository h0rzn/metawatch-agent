package dock

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/client"
)

type Events struct {
	c    *client.Client
	Disp Dispatcher
}

type Dispatcher struct {
	Container chan events.Message
}

func (disp *Dispatcher) Dispatch(evs <-chan events.Message) {
	for event := range evs {
		if event.Type == events.ContainerEventType {
			disp.Container <- event
		}
	}
}

func (ev *Events) listener() (<-chan events.Message, <-chan error) {
	ctx := context.Background()
	opts := types.EventsOptions{}
	return ev.c.Events(ctx, opts)
}

// https://docs.docker.com/engine/reference/commandline/events/
// func (ev *Events) catch(event events.Message) <-chan events.Message {
// 	switch event.Status {
// 	case "create":
// 	case "destroy":
// 	case "die":
// 	case "health_status":
// 	case "pause":
// 	case "rename":
// 	case "restart":
// 	case "start":
// 	case "stop":
// 	case "unpause":
// 	case "update":
// 		break
// 	default:
// 		break
// 	}

// }

func (ev *Events) Run() {

	evs, _ := ev.listener()
	_ = evs
}
