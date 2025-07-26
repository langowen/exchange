package config

import (
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
	"log"
	"reflect"
	"strings"
	"time"
)

type Config struct {
	Storage    Storage
	HTTPServer HTTPServer
	Fetcher    Fetcher
}

type Storage struct {
	Timeout  time.Duration `env:"BD_TIMEOUT" env-default:"10s"`
	Host     string        `env:"BD_HOST" env-required:"true"`
	Port     int           `env:"BD_PORT" env-required:"true"`
	User     string        `env:"BD_USER" env-required:"true"`
	Password string        `env:"BD_PASSWORD" env-required:"true"`
	DBName   string        `env:"BD_DBNAME" env-required:"true"`
	SSLMode  string        `env:"BD_SSL_MODE" env-default:"disable"`
	Schema   string        `env:"BD_SCHEMA" env-default:"dev"`
}

type HTTPServer struct {
	Port        string        `env:"HTTP_PORT" env-default:"8082"`
	Timeout     time.Duration `env:"HTTP_TIMEOUT" env-default:"2m"`
	IdleTimeout time.Duration `env:"HTTP_IDLE_TIMEOUT" env-default:"60s"`
}

type Fetcher struct {
	URL         string        `env:"FETCHER_URL" env-default:"https://min-api.cryptocompare.com/data/price"`
	Rate        string        `env:"FETCHER_RATE" env-default:"BTC,ETH"`
	ValueRate   string        `env:"FETCHER_VALUE_RATE" env-default:"USD,JPY,EUR"`
	Timeout     time.Duration `env:"FETCHER_TIMEOUT" env-default:"10s"`
	TimeTickers time.Duration `env:"FETCHER_TIME_TICKERS" env-default:"10s"`
}

func NewConfig() *Config {
	cfg := &Config{}

	_ = godotenv.Load(".env")

	err := cleanenv.ReadEnv(cfg)
	if err != nil {
		log.Fatal("Error reading env")
	}

	return cfg
}

func (c *Config) Split(fieldName string) []string {
	v := reflect.ValueOf(&c.Fetcher).Elem()
	f := v.FieldByName(fieldName)
	if !f.IsValid() || f.Kind() != reflect.String {
		return nil
	}
	str := f.String()
	if str == "" {
		return nil
	}
	return strings.Split(str, ",")
}
