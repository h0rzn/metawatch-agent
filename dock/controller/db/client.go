package db

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
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
	logrus.Info("- DB::Client - connection successful")

	// ping con

	// defer func() {
	// 	if err := client.Disconnect(context.TODO()); err != nil {
	// 		panic(err)
	// 	}
	// }()
	c.Mongo = client

	return
}

func (c *Client) BulkWrite(data []interface{}) {
	if len(data) == 0 {
		return
	}
	col := c.Mongo.Database("metawatch").Collection("metrics")
	ctx := context.Background()
	res, err := col.InsertMany(ctx, data)
	if err != nil {
		logrus.Errorf("- DB - bulk write err:", err)
		return
	}
	logrus.Infof("- DB - sucessful bulk write of %d entries\n", len(res.InsertedIDs))
	// for _, sRes := range res.InsertedIDs {
	// 	fmt.Printf("%+v\n", sRes)
	// }

	// coll := c.Mongo.Database("metawatch").Collection("metrics")
	//  address1 := Address{"1 Lakewood Way", "Elwood City", "PA"}
	// student1 := Student{FirstName: "Arthur", Address: address1, Age: 8}
	// _, err = coll.InsertOne(context.TODO(), student1)
}
