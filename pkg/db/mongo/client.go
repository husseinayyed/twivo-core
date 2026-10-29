package mongo

import (
	"context"
	"errors"
	"time"

	"github.com/husseinayyed/twivo-core/pkg/breaker"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)


func Connect(MONGO_URL string, MONGO_USER string, MONGO_PASSWORD string) (*mongo.Client, error) {
	if MONGO_URL == "" || MONGO_USER == "" || MONGO_PASSWORD == "" {
		
		return nil, errors.New("missing required environment variables for MongoDB connection")
	}

	result, err := breaker.Group.Execute("Mongo", func() (any, error) {
		credential := options.Credential{
			Username: MONGO_USER,
			Password: MONGO_PASSWORD,
			AuthSource: "admin",
		}

		client, err := mongo.Connect(options.Client().ApplyURI(MONGO_URL).SetAuth(credential))
		if err != nil {
			return nil, err
		}

		pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer pingCancel()

		if err := client.Ping(pingCtx, nil); err != nil {
			_ = client.Disconnect(context.Background())
			return nil, err
		}

		return client, nil
	})
	if err != nil {
		log.Error().Str("service", "MongoDB").Err(err).Msg("failed to connect to MongoDB")
		return nil, err
	}

	client, ok := result.(*mongo.Client)
	if !ok {
		return nil, errors.New("MongoDB breaker returned an invalid client type")
	}

	// 🎉 If execution gets here, the connection is active and ready!
	log.Info().Str("service", "MongoDB").Msg("successfully connected to MongoDB")
	return client, nil
}
