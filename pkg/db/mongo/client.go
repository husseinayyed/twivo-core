package mongo

import (
	"context"
	"errors"
	"time"

	"github.com/husseinayyed/twivo-core/pkg/boostrap"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)


func Connect(MONGO_URL string, MONGO_USER string, MONGO_PASSWORD string) (*mongo.Client, error) {
	if MONGO_URL == "" || MONGO_USER == "" || MONGO_PASSWORD == "" {
		
		return nil, errors.New("missing required environment variables for MongoDB connection")
	}

	var client *mongo.Client
	var err error

	_, err = boostrap.Breaker.Execute("Mongo",func() (any, error) {
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
		log.Service("MongoDB").Err(err).Msg("failed to connect to MongoDB")
	}

	// 🎉 If execution gets here, the connection is active and ready!
	log.Info().Service("MongoDB").Msg("successfully connected to MongoDB")
	
	return client, nil
}
