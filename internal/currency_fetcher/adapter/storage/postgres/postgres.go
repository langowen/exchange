package postgres

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/langowen/exchange/internal/entities"
	"github.com/pkg/errors"
	"time"
)

type Storage struct {
	db *pgxpool.Pool
}

func NewStorage(pool *pgxpool.Pool) *Storage {
	return &Storage{
		db: pool,
	}
}

func InitStorage(ctx context.Context, dsn string) (*Storage, error) {
	const op = "storage.postgres.New"

	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, errors.Wrap(err, op)
	}
	poolConfig.MaxConns = 25
	poolConfig.MinConns = 5
	poolConfig.MaxConnLifetime = 10 * time.Minute
	poolConfig.MaxConnIdleTime = 5 * time.Minute

	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, errors.Wrap(err, op)
	}

	if err = pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, errors.Wrap(err, op)
	}

	storageBD := NewStorage(pool)

	return storageBD, nil
}

func (s *Storage) SaveRates(ctx context.Context, rates []entities.ExchangeRate) error {
	const op = "storage.postgres.SaveRates"

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return errors.Wrap(err, op)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	for _, rate := range rates {
		var cryptoID int

		err = tx.QueryRow(ctx, `SELECT id FROM cryptocurrencies WHERE code = $1`, rate.Title).Scan(&cryptoID)
		if err != nil {
			return errors.Wrap(err, op)
		}

		for _, fiatValue := range rate.FiatValues {
			var fiatID int

			err = tx.QueryRow(ctx, `SELECT id FROM fiat_currencies WHERE code = $1`, fiatValue.Currency).Scan(&fiatID)
			if err != nil {
				return errors.Wrap(err, op)
			}

			_, err = tx.Exec(ctx, `
                INSERT INTO exchange_rates (crypto_id, fiat_id, amount, timestamp)
                VALUES ($1, $2, $3, $4)
                ON CONFLICT (crypto_id, fiat_id, timestamp) 
                DO UPDATE SET amount = EXCLUDED.amount
            `, cryptoID, fiatID, fiatValue.Amount, rate.DateUpdate)
			if err != nil {
				return errors.Wrap(err, op)
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return errors.Wrap(err, op)
	}

	return nil
}

func (s *Storage) GetRates(ctx context.Context) ([]entities.ExchangeRate, error) {
	const op = "storage.postgres.GetRates"

	cryptoQuery := `SELECT code FROM cryptocurrencies ORDER BY id`
	cryptoRows, err := s.db.Query(ctx, cryptoQuery)
	if err != nil {
		return nil, errors.Wrap(err, op)
	}
	defer cryptoRows.Close()

	var cryptoCurrencies []string
	for cryptoRows.Next() {
		var code string
		if err = cryptoRows.Scan(&code); err != nil {
			return nil, errors.Wrap(err, op)
		}
		cryptoCurrencies = append(cryptoCurrencies, code)
	}

	if err := cryptoRows.Err(); err != nil {
		return nil, errors.Wrap(err, op)
	}

	fiatQuery := `SELECT code FROM fiat_currencies ORDER BY id`
	fiatRows, err := s.db.Query(ctx, fiatQuery)
	if err != nil {
		return nil, errors.Wrap(err, op)
	}
	defer fiatRows.Close()

	var fiatCurrencies []entities.FiatPrice
	for fiatRows.Next() {
		var fiat entities.FiatPrice
		if err = fiatRows.Scan(&fiat.Currency); err != nil {
			return nil, errors.Wrap(err, op)
		}
		fiatCurrencies = append(fiatCurrencies, fiat)
	}

	if err = fiatRows.Err(); err != nil {
		return nil, errors.Wrap(err, op)
	}

	result := make([]entities.ExchangeRate, 0, len(cryptoCurrencies))
	for _, cryptoCode := range cryptoCurrencies {
		fiat := make([]entities.FiatPrice, len(fiatCurrencies))
		copy(fiat, fiatCurrencies)

		rate, err := entities.NewRate(cryptoCode, fiat, time.Time{})
		if err != nil {
			return nil, errors.Wrap(err, op)
		}
		result = append(result, *rate)
	}

	return result, nil
}

func (s *Storage) SaveNewCurrency(ctx context.Context, currency string) error {
	const op = "storage.postgres.SaveNewCurrency"

	_, err := s.db.Exec(ctx, `INSERT INTO cryptocurrencies (code) VALUES ($1)`, currency)
	if err != nil {
		return errors.Wrap(err, op)
	}

	return nil
}
