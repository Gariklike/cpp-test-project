package mongo

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoRepository struct {
	client     *mongo.Client
	database   *mongo.Database
	collection *mongo.Collection
}

func NewMongoRepository(uri, dbName, collectionName string) (*MongoRepository, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}

	// Проверка соединения
	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, err
	}

	db := client.Database(dbName)
	collection := db.Collection(collectionName)

	return &MongoRepository{
		client:     client,
		database:   db,
		collection: collection,
	}, nil
}

func (r *MongoRepository) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return r.client.Disconnect(ctx)
}

// Пример метода для логирования
func (r *MongoRepository) LogAuthEvent(event map[string]interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	event["timestamp"] = time.Now()
	_, err := r.collection.InsertOne(ctx, event)
	return err
}
