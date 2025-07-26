package service

import "context"

type HTTPClient interface {
	Fetch(ctx context.Context, url string) (map[string]float64, error)
}

//TODO переделать мапу в сущность
