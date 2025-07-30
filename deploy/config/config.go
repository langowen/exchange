package config

import (
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
	"log"
	"time"
)

type Config struct {
	Storage    Storage
	HTTPServer HTTPServer
	Fetcher    Fetcher
	Redis      Redis
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
	URL         string        `env:"FETCHER_URL" env-default:"https://min-api.cryptocompare.com/data/pricemulti"`
	Timeout     time.Duration `env:"FETCHER_TIMEOUT" env-default:"10s"`
	TimeTickers time.Duration `env:"FETCHER_TIME_TICKERS" env-default:"10s"`
}

type Redis struct {
	Host     string `yaml:"host" env:"REDIS_HOST" env-required:"true"`
	Password string `yaml:"password" env:"REDIS_PASSWORD" env-default:""`
	DB       int    `yaml:"db" env:"REDIS_DB" env-default:"5"`
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
