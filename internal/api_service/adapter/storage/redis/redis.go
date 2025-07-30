package redis

import (
	"context"
	"github.com/langowen/exchange/internal/entities"
	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"
	"log/slog"
	"net"
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
	const op = "storage.redis.InitStorage"

	redisClient := redis.NewClient(options)

	if _, err := redisClient.Ping(ctx).Result(); err != nil {
		return nil, errors.Wrap(err, op)
	}

	storage := NewStorage(redisClient)

	return storage, nil
}

func (s *Storage) ListenUdp(ctx context.Context) (string, error) {
	const op = "storage.redis.ListenUdp"

	pubsub := s.rdb.Subscribe(ctx, "currency_updated")

	msg, err := pubsub.ReceiveMessage(ctx)
	if err != nil {
		var netErr net.Error
		if errors.As(err, &netErr) {
			if netErr.Timeout() {
				return "", entities.ErrRedisTimeout
			}
			return "", entities.ErrRedisCanceled
		}
		return "", errors.Wrap(err, op)
	}

	currency := msg.Payload

	slog.Debug("Received message", "currency", currency)

	return currency, nil
}

func (s *Storage) PublishNew(ctx context.Context, currency string) error {
	s.rdb.Publish(ctx, "new_currency", currency)

	return nil
}
