package db

import (
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/h0rzn/monitoring_agent/dock/metrics"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

const (
	URI    string = "mongodb://root:root@127.0.0.1:27017/"
	DBName string = "metawatch"
)

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

type DB struct {
	Client *mongo.Client
	URI    string
}

func NewDB() *DB {
	return &DB{}
}

func (db *DB) Init() error {
	logrus.Infoln("- DB - init...")

	uri := os.Getenv("DB")
	if uri == "" {
		return errors.New("")
	}
	db.URI = uri

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(db.URI))
	if err != nil {
		return err
	}
	logrus.Info("- DB::Client - connection successful")
	db.Client = client

	err = db.InitScheme()
	return err
}

// InitScheme initiates collections
func (db *DB) InitScheme() error {
	dbc := db.Client.Database("metawatch")

	// metawatch.metrics
	tso := options.TimeSeries().SetTimeField("when").SetMetaField("cid")
	opts := options.CreateCollection().SetTimeSeriesOptions(tso)
	err := dbc.CreateCollection(context.TODO(), "metrics", opts)

	e := &mongo.CommandError{}
	if errors.As(err, e) && e.Code == 48 {
		logrus.Infoln("- DB - metawatch.metrics found")
	} else if err == nil {
		logrus.Infoln("- DB - metawatch.metrics created")
	}

	// metawatch.users
	opts = &options.CreateCollectionOptions{}
	err = dbc.CreateCollection(context.TODO(), "users", opts)

	e = &mongo.CommandError{}
	if errors.As(err, e) && e.Code == 48 {
		logrus.Infoln("- DB - metawatch.users found")
	} else if err == nil {
		logrus.Infoln("- DB - metawatch.users created")
	}

	return nil
}

func (db *DB) User(id primitive.ObjectID) (User, error) {
	var user User
	col := db.Client.Database("metawatch").Collection("users")

	filter := bson.D{{"_id", id}}
	err := col.FindOne(context.TODO(), filter).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return user, errors.New("could not find user")
		}
		return user, err
	}

	return user, nil
}

func (db *DB) InsertUser(u User) (User, error) {
	u.HashPassword()
	u.SetCreated()

	col := db.Client.Database("metawatch").Collection("users")

	filter := bson.D{{"name", u.Name}}
	err := col.FindOne(context.TODO(), filter).Decode(&User{})
	if err != mongo.ErrNoDocuments {
		return User{}, errors.New("user exists already")
	}

	result, err := col.InsertOne(context.TODO(), u)
	if err != nil {
		logrus.Errorf("- DB - users insert err:", err)
		return User{}, errors.New("failed to inser user: " + err.Error())
	}

	logrus.Debugln("- DB - inserted user")

	id, ok := result.InsertedID.(primitive.ObjectID)
	if !ok {
		return User{}, errors.New("error retrieving inserted user")
	}

	user, err := db.User(id)
	if err != nil {
		return user, err
	}

	user.RemovePassword()
	return user, nil
}

func (db *DB) RemoveUser(id string) error {
	col := db.Client.Database("metawatch").Collection("users")

	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("id cant be parsed")
	}

	res, err := col.DeleteOne(context.TODO(), bson.D{{"_id", objID}})
	if err != nil {
		return err
	}
	fmt.Printf("%+v\n", res)

	if res.DeletedCount != 1 {
		return errors.New("failed to delete nonexistent user")
	}

	return nil
}

// + param: update map
func (db *DB) UpdateUser(update map[string]string, id string) (map[string]interface{}, error) {
	var user User
	status := make(map[string]bool)
	result := make(map[string]interface{})

	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("cant parse id")
	}

	filter := bson.D{{"_id", objID}}
	col := db.Client.Database("metawatch").Collection("users")
	err = col.FindOne(context.TODO(), filter).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return result, errors.New("could not find user")
		}
		return result, err
	}
	fmt.Printf("user found: %+v\n", user)

	userFields := reflect.ValueOf(user)
	for key, val := range update {
		fmt.Println("handling", key, val)
		field := userFields.FieldByName(strings.Title(strings.ToLower(key)))
		if field == (reflect.Value{}) {
			fmt.Printf("unkown field %s\n", key)
			status[key] = false
		} else {
			var change bson.D
			if val == "" {
				status[key] = false
				continue
			}

			// protect fields
			switch key {
			case "password":
				user.UpdatePassword(val)
				change = bson.D{{"$set", bson.D{{key, user.Password}}}}
			case "name":
				change = bson.D{{"$set", bson.D{{key, val}}}}
			case "created":
				status[key] = false
				continue
			default:
				status[key] = false
				continue
			}

			patch, err := col.UpdateOne(context.TODO(), filter, change)
			if err == nil && patch.ModifiedCount == 1 {
				fmt.Println("user modified", err, patch)
				status[key] = true
			} else {
				status[key] = false
			}
		}
	}

	updated, err := db.User(objID)
	if err != nil {
		return result, err
	}
	updated.RemovePassword()

	result["user"] = updated
	result["status"] = status

	return result, nil
}

func (db *DB) GetUsers() (result []User, err error) {
	col := db.Client.Database("metawatch").Collection("users")
	cur, err := col.Find(context.TODO(), bson.D{})
	if err != nil {
		return
	}

	for cur.Next(context.TODO()) {
		var u User
		err := cur.Decode(&u)
		if err != nil {
			logrus.Errorf("- DB - failed to get users: %s", err)
		} else {
			u.RemovePassword()
			result = append(result, u)
		}
	}
	return
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

	col := db.Client.Database("metawatch").Collection("metrics")
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

func (db *DB) InsertManyMetrics(data []interface{}) {
	if len(data) == 0 {
		return
	}

	col := db.Client.Database("metawatch").Collection("metrics")
	ctx := context.Background()
	res, err := col.InsertMany(ctx, data)
	if err != nil {
		logrus.Errorf("- DB - bulk write err:", err)
		return
	}
	logrus.Infof("- DB - sucessful insert of %d metric entries\n", len(res.InsertedIDs))
}

func (db *DB) HashByUser(username string) (bool, []byte) {
	var user User
	col := db.Client.Database("metawatch").Collection("users")
	filter := bson.D{{"name", username}}
	err := col.FindOne(context.TODO(), filter).Decode(&user)
	if err == mongo.ErrNoDocuments {
		return false, []byte{}
	}

	return true, []byte(user.Password)
}

func (db *DB) PasswordCorrect(username, password string) bool {
	if exists, hash := db.HashByUser(username); exists {
		return compare(username, hash)
	}
	return false
}

func (db *DB) UserExists(username string) bool {
	users, err := db.GetUsers()
	if err != nil {
		return false
	}

	for _, user := range users {
		if user.Name == username {
			return true
		}
	}
	return false
}

// https://astaxie.gitbooks.io/build-web-application-with-golang/content/en/09.5.html

func hash(input string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(input), bcrypt.DefaultCost)

}

func compare(input string, hash []byte) bool {
	if err := bcrypt.CompareHashAndPassword(hash, []byte(input)); err != nil {
		return false
	}
	return true
}
