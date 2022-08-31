package plugin

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	schemaName     = "crossgate"
	collectionName = "discovery"
	layout         = "2006-01-02 15:04:05"
)

type MongoContent struct {
	Id      string    `json:"_id" bson:"_id"`
	Content *Content  `json:"content" bson:"content"`
	Time    time.Time `json:"time" bson:"time"`
}

var _ Plugin = &Mongo{}

type Mongo struct {
	*MongoContent
	client *mongo.Client
}

func initMongoPlugin(ctx context.Context, wg *sync.WaitGroup, uri string) (Plugin, error) {
	client, err := connect(ctx, uri)
	if err != nil {
		return nil, err
	}
	mongo := &Mongo{MongoContent: nil, client: client}

	wg.Add(1)

	go func() {
		defer func() {
			mongo.unregister()
			wg.Done()
		}()

		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second * 2):
				if mongo.MongoContent == nil {
					continue
				}
				if err := mongo.Set(ctx, "", *mongo.Content); err != nil {
					fmt.Printf("renewal error %s", err)
				}
			}
		}
	}()

	return mongo, nil
}

func (m *Mongo) unregister() error {
	if m.MongoContent == nil {
		return nil
	}

	id, _ := primitive.ObjectIDFromHex(m.MongoContent.Id)

	_, err := m.client.
		Database(schemaName).
		Collection(collectionName).
		DeleteOne(context.Background(), bson.M{"_id": id})

	return err
}

func (m *Mongo) Set(ctx context.Context, name string, value Content) error {
	mc := MongoContent{}

	upsert := true
	filter := bson.D{
		{Key: "content.service", Value: value.Service},
		{Key: "content.addr", Value: value.Addr},
	}

	res := m.client.Database(schemaName).
		Collection(collectionName).FindOne(ctx, filter)

	if res.Err() == mongo.ErrNoDocuments {
		mc.Id = primitive.NewObjectID().String()
		mc.Content = &value
	} else {
		if err := res.Decode(&mc); err != nil {
			return err
		}
	}

	m.MongoContent = &mc

	update := bson.D{
		{Key: "$set", Value: bson.D{
			{Key: "content.service", Value: value.Service},
			{Key: "content.lba", Value: value.Lba},
			{Key: "content.addr", Value: value.Addr},
			{Key: "time", Value: time.Now()},
		}}}
	_, err := m.client.
		Database(schemaName).
		Collection(collectionName).
		UpdateOne(ctx,
			filter,
			update,
			&options.UpdateOptions{
				Upsert: &upsert,
			})

	if err != nil {
		return err
	}

	return nil
}

func connect(ctx context.Context, uri string) (*mongo.Client, error) {
	cliOpt := options.Client()
	cliOpt.SetMaxPoolSize(2000)
	cliOpt.SetMinPoolSize(1)
	cliOpt.SetMaxConnIdleTime(time.Second)

	cliOpt.SetRegistry(
		bson.NewRegistryBuilder().
			RegisterTypeMapEntry(
				bsontype.DateTime,
				reflect.TypeOf(time.Time{})).
			Build(),
	)

	cliOpt.ApplyURI(uri)

	cli, err := mongo.NewClient(cliOpt)
	if err != nil {
		return nil, err
	}

	if err := cli.Connect(ctx); err != nil {
		return nil, err
	}

	if err := cli.Ping(ctx, nil); err != nil {
		return nil, err
	}

	return cli, nil
}
