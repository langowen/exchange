package entities

import "errors"

//TODO типичные ошибки. Для конструкторов, сервиса

var (
	ErrNotFound = errors.New("entity not found")
)

//TODO завернуть все ошибки https://github.com/pkg/errors
