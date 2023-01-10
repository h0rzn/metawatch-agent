package db

import (
	"context"
	"fmt"

	"github.com/h0rzn/monitoring_agent/dock/metrics"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
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
	logrus.Infoln("- DB - init...")
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

	//db.Metrics("e63d12ed9e74cb2d5994e9e0356aad7588939108fb2a1e5ec729274e07e820bf")
	return nil

}

func (db *DB) Metrics(cid string, tmin primitive.DateTime, tmax primitive.DateTime) map[string][]metrics.Set {
	logrus.Debugf("- DB - aggregate for %s (%s-%s)\n", cid, tmin.Time(), tmax.Time())

	match := bson.D{
		{Key: "$match",
			Value: bson.D{
				{Key: "cid", Value: cid},
				{Key: "when", Value: bson.D{
					{Key: "$gte", Value: tmin},
					{Key: "$lte", Value: tmax},
				}},
			},
		},
	}

	// add group stage to only get $when and $metrics

	col := db.Client.Mongo.Database("metawatch").Collection("metrics")
	ctx := context.Background()
	curs, err := col.Aggregate(ctx, mongo.Pipeline{match})

	if err != nil {
		logrus.Errorf("- DB - metrics aggregation err: %s", err)
	}
	var result []MetricsMod
	if err = curs.All(ctx, &result); err != nil {
		logrus.Errorf("- DB - metrics aggregation err (getting all from cursor): %s", err)
	}

	out := make(map[string][]metrics.Set)
	out[cid] = make([]metrics.Set, 0)
	logrus.Debugf("- DB - aggregated %d sets\n", len(result))

	for _, prim := range result {
		set := prim.Metrics
		set.When = prim.When
		out[cid] = append(out[cid], set)
	}

	return out
}

type MetricsMod struct {
	MongoID primitive.ObjectID `bson:"_id,omitempty"`
	CID     string             `bson:"cid"`     // metadata field
	When    primitive.DateTime `bson:"when"`    // time
	Metrics metrics.Set        `bson:"metrics"` // actual data
}

func NewMetricsMod(cid string, when primitive.DateTime, metrics metrics.Set) *MetricsMod {
	return &MetricsMod{
		CID:     cid,
		When:    when,
		Metrics: metrics,
	}
}
