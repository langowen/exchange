package fetcher

import "context"

type RedisStorage interface {
	PublishUpd(ctx context.Context, currency string) error
	ListenNew(ctx context.Context) (string, error)
}
