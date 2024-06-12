package db

import (
	"context"
	"errors"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var MgoCli *mongo.Client
var collection *mongo.Collection

const LimitConversactionMsg = 20

var MongoURI = "mongodb://admin:admin@127.0.0.1:27017"

func Init() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var err error
	MgoCli, err = mongo.Connect(ctx, options.Client().ApplyURI(MongoURI))
	if err != nil {
		log.Panic("connect mongo error", err)
	}
	err = MgoCli.Ping(context.TODO(), nil)
	if err != nil {
		log.Panic("ping mongo error", err)
	}
	collection = MgoCli.Database("aos").Collection("conversation")
}

func InsertSingleConversation(msg Message) error {
	ctx, _ := context.WithTimeout(context.Background(), time.Minute)
	if _, err := collection.InsertOne(ctx, msg); err != nil {
		return err
	}
	return nil
}

func GetResentConversation(conversationId string, startTime int64) ([]Message, error) {
	if conversationId == "" {
		return nil, errors.New("conversation id empty")
	}
	opt := options.Find().SetSort(bson.D{{Key: "startTime", Value: -1}}).SetLimit(LimitConversactionMsg)
	filter := bson.D{{Key: "conversationId", Value: conversationId}, {Key: "startTime", Value: bson.D{{Key: "$gt", Value: startTime}}}}
	cursor, err := collection.Find(context.TODO(), filter, opt)
	if err != nil {
		return nil, err
	}
	msgLog := make([]Message, 0)
	for cursor.Next(context.TODO()) {
		var msg Message
		if err := cursor.Decode(&msg); err != nil {
			continue
		}
		msgLog = append(msgLog, msg)
	}
	return msgLog, nil
}
