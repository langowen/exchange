package redis

import (
	"context"
	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"
	"log/slog"
)

type Storage struct {
	rdb *redis.Client
}

func NewStorage(client redis.UniversalClient) *Storage {
	return &Storage{
		rdb: client.(*redis.Client),
	}
}

func InitStorage(ctx context.Context, options *redis.Options) (*Storage, error) {
	redisClient := redis.NewClient(options)

	if _, err := redisClient.Ping(ctx).Result(); err != nil {
		return nil, err
	}

	storage := NewStorage(redisClient)

	return storage, nil
}

func (s *Storage) ListenNew(ctx context.Context) (string, error) {
	const op = "redis.ListenNew"
	pubsub := s.rdb.Subscribe(ctx, "new_currency")

	msg, err := pubsub.ReceiveMessage(ctx)
	if err != nil {
		return "", errors.Wrap(err, op)
	}

	currency := msg.Payload

	slog.Debug("Received message", "currency", currency)

	return currency, nil
}

func (s *Storage) PublishUpd(ctx context.Context, currency string) error {
	s.rdb.Publish(ctx, "currency_updated", currency)

	return nil
}
