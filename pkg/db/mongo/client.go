package mongodb

import (
	"context"
	"github.com/husseinayyed/twivo-core/pkg/breaker"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"os"
	"time"
)

var (
	MONGO_URL      = os.Getenv("MONGO_URL")
	MONGO_USER     = os.Getenv("MONGO_USER")
	MONGO_PASSWORD = os.Getenv("MONGO_PASSWORD")
	Client           *mongo.Client
)

func Connect() {
	if MONGO_URL == "" || MONGO_USER == "" || MONGO_PASSWORD == "" {
		log.Fatal().Msg("one or more of MONGO_URL, MONGO_USER, MONGO_PASSWORD environment variables must be set")
	}

	var client *mongo.Client
	var err error

	_, err = breaker.MongoDB.Execute(func() (any, error) {
		credential := options.Credential{
			Username: MONGO_USER,
			Password: MONGO_PASSWORD,
		}

		client, err = mongo.Connect(options.Client().ApplyURI(MONGO_URL).SetAuth(credential))
		if err != nil {
			return nil, err
		}

		pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer pingCancel()

		if err = client.Ping(pingCtx, nil); err != nil {
			_ = client.Disconnect(context.Background())
			return nil, err
		}

		return client, nil
	})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to MongoDB")
	}

	// 🎉 If execution gets here, the connection is active and ready!
	log.Info().Msg("successfully connected to MongoDB")
	Client = client

}
