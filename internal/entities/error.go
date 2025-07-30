package entities

import "errors"

//TODO типичные ошибки. Для конструкторов, сервиса

var (
	ErrNotFound      = errors.New("entity not found")
	ErrRedisTimeout  = errors.New("timeout waiting for Redis message")
	ErrRedisCanceled = errors.New("redis subscription canceled")
)

//TODO завернуть все ошибки https://github.com/pkg/errors
