package db

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Client struct {
	Mongo *mongo.Client
	URI   string
}

func NewClient() *Client {
	return &Client{}
}

func (c *Client) Init() (err error) {
	// init redis
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(c.URI))
	if err != nil {
		return
	}
	// defer func() {
	// 	if err := client.Disconnect(context.TODO()); err != nil {
	// 		panic(err)
	// 	}
	// }()
	c.Mongo = client

	return
}

func (c *Client) Write(data []WriteSet) {
	fmt.Println("-----db.write-----")
	for i, wr := range data {
		// fmt.Println(wr.ContainerID, wr.Metrics)
		fmt.Printf("[CLIENT] collected [%d] %s: %s", i, wr.ContainerID, string(wr.Metrics))
	}
	fmt.Println("------------------")
}

type WriteSet struct {
	ContainerID string
	Metrics     []byte
}
