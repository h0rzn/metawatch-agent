package hub

import (
	"errors"
	"fmt"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
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
	Done       chan struct{}
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
		Done:       make(chan struct{}),
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
	go c.dpatch(input)
	<-c.Done

	done <- struct{}{}
	c.Leave()
}

func (c *Client) fetch(con *websocket.Conn, done chan struct{}) <-chan *RequestFrame {
	out := make(chan *RequestFrame)
	go func() {
		for {
			select {
			case <-done:
				logrus.Debugln("- CLIENT - received done signal")
				close(out)
				return
			default:
			}

			var frame *RequestFrame
			err := c.con.ReadJSON(&frame)
			if frame == nil {
				logrus.Infoln("- CLIENT - received empty frame, later!")
				break
			}
			fmt.Println("[CLIENT::fetch]frame:", frame)
			if err != nil {
				if !websocket.IsCloseError(err) && !websocket.IsUnexpectedCloseError(err) {
					logrus.Errorf("- CLIENT - error reading json message: %s\n", err)
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
			logrus.Errorf("- CLIENT - unkown ressource type %s\n", frame.Type)
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
			logrus.Errorf("- CLIENT - unkown event type: %s\n", frame.Event)
			// send error message: unkown event type
		}
	}
}

func (c *Client) Input() {
	fmt.Println("client input range")
	for frame := range c.In {
		c.Send(frame)
	}
	fmt.Println("client input ranged ended")
}

func (c *Client) Leave() {
	fmt.Println("[CLIENT] Leaving...")
	c.Eps.Leave <- c
	c.mutex.Lock()
	c.con.Close()
	c.mutex.Unlock()
}

func (c *Client) ForceLeave() {
	logrus.Debugln("- CLIENT - leaving by force...")
	c.mutex.Lock()
	// send later message?
	c.con.Close()
	c.Done <- struct{}{}
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
