package hub

import (
	"context"
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
	CID     string      `json:"container_id,omitempty"`
	Type    string      `json:"type"`
	Message interface{} `json:"message"`
}

type CloseMessage struct {
	Type string `json:"type"`
}

type Demand struct {
	Client    *Client
	CID       string
	Ressource string
}

type Client struct {
	wg       *sync.WaitGroup
	ctx      context.Context
	cancel   context.CancelFunc
	con      *websocket.Conn
	In       chan *Response
	Sub      chan *Demand
	USub     chan *Demand
	Lve      chan *Client
	sndClose chan CloseMessage
}

func NewClient(con *websocket.Conn, sub chan *Demand, usub chan *Demand, lve chan *Client) *Client {
	return &Client{
		wg:       &sync.WaitGroup{},
		con:      con,
		In:       make(chan *Response),
		Sub:      sub,
		USub:     usub,
		Lve:      lve,
		sndClose: make(chan CloseMessage, 1),
	}
}

func (c *Client) parse() {
	defer func() {
		c.wg.Done()
		fmt.Println("CLIENT parse done")
	}()

	for {
		select {
		case <-c.ctx.Done():
			fmt.Println("parse done <-c.done")
			return
		default:
			var frame *Request
			err := c.con.ReadJSON(&frame)
			fmt.Println(err, frame)
			if err != nil {
				go c.CloseByRemote()
				select {
				case <-c.ctx.Done():
					return
				case <-time.After(3 * time.Second):
					fmt.Println("CLIENT parse failed to finish")
				}
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

func (c *Client) HandleSend() {
	defer func() {
		c.wg.Done()
	}()
	for {
		select {
		case <-c.ctx.Done():
			return
		case closeMsg := <-c.sndClose:
			_ = c.con.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, closeMsg.Type), time.Now().Add(3*time.Second))
			c.con.Close()
		case response := <-c.In:
			err := c.con.WriteJSON(response)
			if err != nil {
				return
			}
		}
	}
}

func (c *Client) CloseByRemote() {
	logrus.Infoln("- CLIENT - closing by remote")
	c.cancel()
	c.wg.Wait()
	c.Lve <- c
	logrus.Debugln("- CLIENT - closed now")
}

func (c *Client) Close() {
	logrus.Infoln("- CLIENT - closing")

	c.sndClose <- CloseMessage{
		Type: "close",
	}
	c.cancel()
	c.wg.Wait()

	c.Lve <- c
	logrus.Debugln("- CLIENT - closed now")
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
	ctx, cancel := context.WithCancel(context.Background())
	c.ctx = ctx
	c.cancel = cancel

	c.wg.Add(2)
	go c.parse()
	go c.HandleSend()
}
