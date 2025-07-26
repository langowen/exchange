CREATE TABLE cryptocurrencies (
                                  id SERIAL PRIMARY KEY,
                                  code VARCHAR(5) UNIQUE NOT NULL
);

CREATE TABLE fiat_currencies (
                                 id SERIAL PRIMARY KEY,
                                 code VARCHAR(5) UNIQUE NOT NULL
);

//TODO убрать ID
CREATE TABLE exchange_rates (
                                id SERIAL PRIMARY KEY,
                                crypto_id INTEGER REFERENCES cryptocurrencies(id) ON DELETE CASCADE,
                                fiat_id INTEGER REFERENCES fiat_currencies(id) ON DELETE CASCADE,
                                amount DECIMAL(20, 2) NOT NULL,
                                timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),

                                CONSTRAINT unique_rate_pair_time UNIQUE (crypto_id, fiat_id, timestamp)
);

CREATE INDEX idx_exchange_rates_crypto ON exchange_rates(crypto_id);
CREATE INDEX idx_exchange_rates_fiat ON exchange_rates(fiat_id);
CREATE INDEX idx_exchange_rates_timestamp ON exchange_rates(timestamp);