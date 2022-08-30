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
	context := context.TODO()

	client, err := connect(context, uri)
	if err != nil {
		return nil, err
	}
	mongo := &Mongo{MongoContent: nil, client: client}

	wg.Add(1)
	go func() {
		defer wg.Done()
		tick := time.NewTicker(2 * time.Second)
		for {
			select {
			case <-ctx.Done():
				if err := mongo.unregister(context); err != nil {
					// ignore
				}
				return
			case <-tick.C:
				if mongo.MongoContent == nil {
					continue
				}
				if err := mongo.Set(context, "", *mongo.Content); err != nil {
					fmt.Printf("renewal error %s", err)
				}
			}
		}
	}()

	return mongo, nil
}

func (m *Mongo) unregister(ctx context.Context) error {
	id, _ := primitive.ObjectIDFromHex(m.MongoContent.Id)
	filter := bson.M{"_id": id}
	_, err := m.client.
		Database(schemaName).
		Collection(collectionName).
		DeleteOne(ctx, filter)
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

	m.MongoContent = &mc

	return nil
}

func getCtx(client *mongo.Client) (context.Context, context.CancelFunc, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	if err := client.Connect(ctx); err != nil {
		return nil, cancel, err
	}
	return ctx, cancel, nil
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
	mcli, err := mongo.NewClient(cliOpt)
	if err != nil {
		return nil, err
	}
	ctx, cancel, err := getCtx(mcli)
	defer cancel()
	if err != nil {
		return nil, err
	}
	if err := mcli.Ping(ctx, nil); err != nil {
		return nil, err
	}
	return mcli, nil
}
