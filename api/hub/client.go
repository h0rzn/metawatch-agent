package hub

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

type Request struct {
	CID   string `json:"container_id"`
	Event string `json:"event"` // eg subscribe
	Type  string `json:"type"`  // eg metrics
}

type Response struct {
	CID     string      `json:"container_id"`
	Type    string      `json:"type"`
	Message interface{} `json:"message"`
}

type Demand struct {
	Client    *Client
	CID       string
	Ressource string
}

type Client struct {
	con     *websocket.Conn
	In      chan *Response
	Sub     chan *Demand
	USub    chan *Demand
	Lve     chan *Client
	done    chan struct{}
	closed  chan struct{}
	closing bool
}

func NewClient(con *websocket.Conn, sub chan *Demand, usub chan *Demand, lve chan *Client) *Client {
	return &Client{
		con:     con,
		In:      make(chan *Response),
		Sub:     sub,
		USub:    usub,
		Lve:     lve,
		done:    make(chan struct{}, 1),
		closed:  make(chan struct{}, 1),
		closing: false,
	}
}

func (c *Client) parse(ctx context.Context, wg *sync.WaitGroup) {
	defer func() {
		wg.Done()
	}()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("parse done <-c.done")
			return
		default:
			var frame *Request
			err := c.con.ReadJSON(&frame)
			if err != nil {
				fmt.Println("parse done (read)")
				if !c.closing {
					fmt.Println("client->c.Lve sig")
					c.Lve <- c
				} else {
					fmt.Println("already closing")
				}
				return
			}

			demand := &Demand{
				CID:       frame.CID,
				Client:    c,
				Ressource: frame.Type,
			}

			switch frame.Event {
			case "subscribe":
				c.Sub <- demand
			case "unsubscribe":
				c.USub <- demand
			default:
				fmt.Println("client unkown event:", frame.Event)
				continue
			}

		}

	}
}

func (c *Client) HandleSend(ctx context.Context, wg *sync.WaitGroup) {
	defer func() {
		wg.Done()
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case response := <-c.In:
			err := c.con.WriteJSON(response)
			if err != nil {
				return
			}
		}
	}
}

func (c *Client) Close() error {
	logrus.Infoln("- CLIENT - closing")
	c.closing = true
	c.done <- struct{}{}

	cls := map[string]string{
		"type": "close",
	}
	_ = c.con.WriteJSON(cls)

	c.con.Close()

	select {
	case <-c.closed:
		return nil
	case <-time.After(25 * time.Second):
		return errors.New("failed to exit goroutines")
	}

}

func (c *Client) Error(msg string) {
	response := &Response{
		CID:     "",
		Type:    "error",
		Message: msg,
	}

	_ = c.con.WriteJSON(response)
}

func (c *Client) Run() {
	wg := sync.WaitGroup{}
	ctx, cancel := context.WithCancel(context.Background())

	wg.Add(2)
	go c.parse(ctx, &wg)
	go c.HandleSend(ctx, &wg)

	<-c.done

	cancel()
	wg.Wait()
	c.closed <- struct{}{}
}
