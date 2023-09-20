package global

import (
	"github.com/Troublor/erebus-redgiant/config"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	mongoClient *mongo.Client
)

func MongoDbClient() *mongo.Client {
	if mongoClient != nil {
		return mongoClient
	}

	var err error
	mongoUrl := viper.GetString(config.CMongoURL.Key)
	opts := options.Client().ApplyURI(mongoUrl)
	mongoClient, err = mongo.Connect(Ctx(), opts)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to MongoDB")
	}
	RegisterCleanupTask(func() {
		err = mongoClient.Disconnect(Ctx())
		if err != nil {
			log.Error().Err(err).Msg("Failed to disconnect from MongoDB")
		}
	})
	return mongoClient
}
