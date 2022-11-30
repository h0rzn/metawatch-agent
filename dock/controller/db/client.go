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

func NewClient(uri string) *Client {
	return &Client{
		URI: uri,
	}
}

func (c *Client) Init() (err error) {
	// init redis
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(c.URI))
	if err != nil {
		return
	}
	fmt.Println("[DB::Client] connection successful")
	// defer func() {
	// 	if err := client.Disconnect(context.TODO()); err != nil {
	// 		panic(err)
	// 	}
	// }()
	c.Mongo = client

	return
}

func (c *Client) BulkWrite(data []interface{}) {
	col := c.Mongo.Database("metawatch").Collection("metrics")
	ctx := context.Background()
	res, err := col.InsertMany(ctx, data)
	if err != nil {
		fmt.Println("[DB] bulk write err:", err)
	}
	fmt.Println("[DB] bulk write:")
	for _, sRes := range res.InsertedIDs {
		fmt.Printf("%+v\n", sRes)
	}

	// coll := c.Mongo.Database("metawatch").Collection("metrics")
	//  address1 := Address{"1 Lakewood Way", "Elwood City", "PA"}
	// student1 := Student{FirstName: "Arthur", Address: address1, Age: 8}
	// _, err = coll.InsertOne(context.TODO(), student1)
}
