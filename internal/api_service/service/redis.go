package service

import "context"

type RedisStorage interface {
	ListenUdp(ctx context.Context) (string, error)
	PublishNew(ctx context.Context, currency string) error
}
