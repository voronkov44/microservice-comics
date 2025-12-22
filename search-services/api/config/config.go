package config

import (
	"log"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type HTTPConfig struct {
	Address         string        `yaml:"address" env:"API_ADDRESS" env-default:"localhost:80"`
	InternalAddress string        `yaml:"internal_address" env:"API_INTERNAL_ADDRESS" env-default:"localhost:81"`
	Timeout         time.Duration `yaml:"timeout" env:"API_TIMEOUT" env-default:"5s"`
}

type Config struct {
	LogLevel         string     `yaml:"log_level" env:"LOG_LEVEL" env-default:"DEBUG"`
	HTTPConfig       HTTPConfig `yaml:"api_server"`
	WordsAddress     string     `yaml:"words_address" env:"WORDS_ADDRESS" env-default:"words:81"`
	UpdateAddress    string     `yaml:"update_address" env:"UPDATE_ADDRESS" env-default:"update:82"`
	SearchAddress    string     `yaml:"search_address" env:"SEARCH_ADDRESS" env-default:"search:83"`
	AuthAddress      string     `yaml:"auth_address"   env:"AUTH_ADDRESS"   env-default:"auth:84"`
	FavoritesAddress string     `yaml:"favorites_address" env:"FAVORITES_ADDRESS" env-default:"favorites:85"`

	// admin jwt verify
	AdminUser     string        `yaml:"admin_user" env:"ADMIN_USER" env-required:"true"`
	AdminPassword string        `yaml:"admin_password" env:"ADMIN_PASSWORD" env-required:"true"`
	TokenTTL      time.Duration `yaml:"token_ttl" env:"TOKEN_TTL" env-default:"2m"`

	// user jwt verify
	AuthJWTSecret string `yaml:"auth_jwt_secret" env:"AUTH_JWT_SECRET" env-required:"true"`

	SearchConcurrency int `yaml:"search_concurrency" env:"SEARCH_CONCURRENCY" env-default:"10"`
	SearchRate        int `yaml:"search_rate"        env:"SEARCH_RATE"        env-default:"100"`
}

func MustLoad(configPath string) Config {
	var cfg Config
	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		log.Fatalf("cannot read config %q: %s", configPath, err)
	}
	return cfg
}
