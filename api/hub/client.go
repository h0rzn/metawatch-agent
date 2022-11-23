package hub

import (
	"errors"
	"fmt"
	"sync"

	"github.com/gorilla/websocket"
)

// RequestFrame is a message holder for client messages/requests
type RequestFrame struct {
	CID   string `json:"container_id"`
	Event string `json:"event"`
	Type  string `json:"type"`
}

// ResponseFrame is a message holder for server responses
type ResponseFrame struct {
	CID     string      `json:"container_id"`
	Type    string      `json:"type"`
	Content interface{} `json:"message"`
}

type Client struct {
	mutex      *sync.Mutex
	con        *websocket.Conn
	Eps        *Endpoints
	In         chan *ResponseFrame
	AllowedRes []string
}

func NewClient(con *websocket.Conn, eps *Endpoints) *Client {
	fmt.Println("creating client")
	allowed := make([]string, 2)
	allowed[0] = "metrics"
	allowed[1] = "logs"

	return &Client{
		mutex:      &sync.Mutex{},
		con:        con,
		Eps:        eps,
		In:         make(chan *ResponseFrame),
		AllowedRes: allowed,
	}
}

// Demand represents a request for subscribing/unsibscribing for a client
type Demand struct {
	Client    *Client
	CID       string
	Ressource string
}

func (c *Client) Handle() {
	go c.Input()
	done := make(chan struct{})
	input := c.fetch(c.con, done)
	c.dpatch(input)
	c.Leave()
}

func (c *Client) fetch(con *websocket.Conn, done chan struct{}) <-chan *RequestFrame {
	out := make(chan *RequestFrame)
	go func() {
		for {
			select {
			case <-done:
				return
			default:
			}

			var frame *RequestFrame
			err := c.con.ReadJSON(&frame)
			if frame == nil {
				fmt.Println("frame is nil, later")
				break
			}
			fmt.Println("frame:", frame)
			if err != nil {
				if !websocket.IsCloseError(err) && !websocket.IsUnexpectedCloseError(err) {
					fmt.Println("error reading json message", err)
				}
				c.SendErr(errors.New("malformed message"))
				break
			} else {
				out <- frame
			}
		}
		close(out)
	}()
	return out
}

func (c *Client) dpatch(in <-chan *RequestFrame) {
	for frame := range in {
		// validate ressource type
		resOK := false
		for _, res := range c.AllowedRes {
			if frame.Type == res {
				resOK = true
			}
		}
		if !resOK {
			fmt.Println("unkown ressource type", frame.Type)
			// send error message
		}

		// create hub demand
		request := &Demand{
			CID:       frame.CID,
			Client:    c,
			Ressource: frame.Type,
		}

		// validate event type
		switch frame.Event {
		case "subscribe":
			c.Eps.Subscribe <- request
		case "unsubscribe":
			c.Eps.Unsubscribe <- request
		default:
			fmt.Println("unkown event type", frame.Event)
			// send error message: unkown event type
		}
	}
}

func (c *Client) Input() {
	for frame := range c.In {
		c.Send(frame)
	}
}

func (c *Client) Leave() {
	fmt.Println("[CLIENT] Leaving...")
	c.Eps.Leave <- c
	c.mutex.Lock()
	c.con.Close()
	c.mutex.Unlock()
}

// SendErr sends an error message to remote con
func (c *Client) SendErr(err error) {
	msg := map[string]string{
		"event":   "error",
		"message": err.Error(),
	}
	c.mutex.Lock()
	err = c.con.WriteJSON(msg)
	c.mutex.Unlock()
	if err != nil {
		c.Leave()
	}
}

// Send sends a normal message to remote con
func (c *Client) Send(rf *ResponseFrame) {
	c.mutex.Lock()
	err := c.con.WriteJSON(rf)
	c.mutex.Unlock()
	if err != nil {
		c.Leave()
	}
}
