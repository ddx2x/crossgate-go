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
	Service string    `json:"service" bson:"service"`
	Lba     string    `json:"lba" bson:"lba"`
	Addr    string    `json:"addr" bson:"addr"`
	Time    time.Time `json:"time" bson:"time"`
	Type    uint8     `json:"type" bson:"type"`
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
	_, err = client.Database(schemaName).
		Collection(collectionName).
		Indexes().
		CreateOne(ctx, mongo.IndexModel{
			Keys:    bson.M{"time": 1},
			Options: options.Index().SetExpireAfterSeconds(2),
		}, &options.CreateIndexesOptions{})
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
				c := Content{
					Service: mongo.Service,
					Addr:    mongo.Addr,
					Lba:     mongo.Lba,
					Type:    mongo.Type,
				}
				if err := mongo.Set(ctx, "", c); err != nil {
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

	_, err := m.client.
		Database(schemaName).
		Collection(collectionName).
		DeleteOne(context.Background(), bson.M{"_id": m.MongoContent.Id})

	return err
}

func (m *Mongo) Set(ctx context.Context, name string, value Content) error {
	mc := MongoContent{
		Id:      primitive.NewObjectID().Hex(),
		Service: value.Service,
		Addr:    value.Addr,
		Lba:     value.Lba,
		Type:    value.Type,
	}

	upsert := true
	filter := bson.D{
		{Key: "service", Value: value.Service},
		{Key: "addr", Value: value.Addr},
	}

	res := m.client.Database(schemaName).
		Collection(collectionName).FindOne(ctx, filter)

	if res.Err() == mongo.ErrNoDocuments {
		mc.Time = time.Now()
	} else {
		var existingDoc MongoContent
		if err := res.Decode(&existingDoc); err != nil {
			return err
		}
		existingDoc.Type = value.Type
		mc = existingDoc
	}

	m.MongoContent = &mc
	// 修改更新语句
	update := bson.M{
		"$set": bson.M{
			"_id":     mc.Id,
			"service": value.Service,
			"lba":     value.Lba,
			"addr":    value.Addr,
			"time":    time.Now(),
			"type":    value.Type,
		},
	}

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
