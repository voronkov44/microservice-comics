package config

import (
	"log"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	LogLevel  string        `yaml:"log_level" env:"LOG_LEVEL" env-default:"DEBUG"`
	Address   string        `yaml:"auth_address" env:"AUTH_ADDRESS" env-default:"localhost:80"`
	DBAddress string        `yaml:"db_address" env:"DB_ADDRESS" env-default:"localhost:82"`
	JWTSecret string        `yaml:"jwt_secret" env:"AUTH_JWT_SECRET" env-required:"true"`
	TokenTTL  time.Duration `yaml:"token_ttl" env:"TOKEN_TTL" env-default:"24h"`
}

func MustLoad(configPath string) Config {
	var cfg Config
	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		log.Fatalf("cannot read config %q: %s", configPath, err)
	}
	return cfg
}
