package db

import (
	"context"
	"fmt"

	"github.com/h0rzn/monitoring_agent/dock/metrics"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	URI    string = "mongodb://root:root@localhost:27017/"
	DBName string = "metawatch"
)

type DB struct {
	Client *Client
}

func NewDB() *DB {
	return &DB{}
}

func (db *DB) Init() error {
	fmt.Println("[DB] init...")
	db.Client = NewClient(URI)
	err := db.Client.Init()
	if err != nil {
		fmt.Println(err)
	}

	// check for errors

	err = db.InitScheme()
	return err
}

// InitScheme initiates collections
func (db *DB) InitScheme() error {
	dbc := db.Client.Mongo.Database("metawatch")
	tso := options.TimeSeries().SetTimeField("when").SetMetaField("cid")
	opts := options.CreateCollection().SetTimeSeriesOptions(tso)
	dbc.CreateCollection(context.TODO(), "metrics", opts)

	return nil

}

type MetricsMod struct {
	CID     string             `bson:"cid"`            // metadata field
	When    primitive.DateTime `bson:"when"`           // time
	Metrics metrics.Set        `bson:"metrics,inline"` // actual data
}

func NewMetricsMod(cid string, when primitive.DateTime, metrics metrics.Set) *MetricsMod {
	return &MetricsMod{
		CID:     cid,
		When:    when,
		Metrics: metrics,
	}
}
