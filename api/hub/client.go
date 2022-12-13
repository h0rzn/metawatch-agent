package hub

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

var allowedTypes = [2]string{"metrics", "logs"}

// RequestFrame is a message holder for client messages/requests
type RequestFrame struct {
	CID   string `json:"container_id"`
	Event string `json:"event"` // eg subscribe
	Type  string `json:"type"`  // eg metrics
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
	inputDone  chan struct{}
	fetchDone  chan struct{}
	finished   chan struct{}
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
		inputDone:  make(chan struct{}),
		fetchDone:  make(chan struct{}),
		finished:   make(chan struct{}),
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

	input := c.fetch(c.con)
	go c.dpatch(input)
	logrus.Infoln("- CLIENT - handling...")
}

func (c *Client) Close() error {
	logrus.Infoln("- CLIENT - closing...")
	c.fetchDone <- struct{}{}
	c.inputDone <- struct{}{}

	select {
	case <-c.finished:
		logrus.Debugln("- CLIENT - finished closing")
		return nil
	case <-time.After(8 * time.Second):
		return errors.New("timeout: pipeline couldnt finish")
	}
}

func (c *Client) typOK(typ string) (found bool) {
	for _, allowed := range allowedTypes {
		if allowed == typ {
			return true
		}
	}
	return
}

func (c *Client) fetch(con *websocket.Conn) <-chan *RequestFrame {
	out := make(chan *RequestFrame)
	go func() {
		for {
			select {
			case <-c.fetchDone:
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
	}()
	return out
}

func (c *Client) dpatch(in <-chan *RequestFrame) {
	for frame := range in {
		if !c.typOK(frame.Type) {
			logrus.Errorf("- CLIENT - unkown ressource type %s\n", frame.Type)
			// just continue for now
			continue
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
	fmt.Println("dpatch finished")
}

func (c *Client) Input() {
	fmt.Println("client input range")
	for {
		select {
		case <-c.inputDone:
			fmt.Println("client: closing input")
			return
		case frame, ok := <-c.In:
			if !ok {
				c.finished <- struct{}{}
				return
			}
			c.Send(frame)
		}
	}
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
