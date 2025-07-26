package entities

import "time"

type ExchangeRate struct {
	Title      string
	FiatValues []FiatPrice
	DateUpdate time.Time
}

type FiatPrice struct {
	Currency string
	Amount   float64
}

func NewRate(title string, values []FiatPrice, date time.Time) (*ExchangeRate, error) {
	rate := &ExchangeRate{
		Title:      title,
		FiatValues: values,
		DateUpdate: date,
	}
	return rate, nil
}
