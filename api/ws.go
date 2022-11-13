package api

import (
	"github.com/gorilla/websocket"
	"github.com/h0rzn/monitoring_agent/dock/stream"
)

func WriteSets(con *websocket.Conn, sets chan *stream.Set, done chan interface{}) {
	for set := range sets {
		err := con.WriteJSON(set)
		if err != nil {
			con.Close()
			done <- struct{}{}
			return
		}
	}
	done <- struct{}{}
	con.Close()

}
