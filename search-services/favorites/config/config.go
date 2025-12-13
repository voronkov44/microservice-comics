package config

import (
	"github.com/ilyakaznacheev/cleanenv"
	"log"
)

type Config struct {
	LogLevel  string `yaml:"log_level" env:"LOG_LEVEL" env-default:"DEBUG"`
	Address   string `yaml:"favorites_address" env:"FAVORITES_ADDRESS" env-default:"localhost:80"`
	DBAddress string `yaml:"db_address" env:"DB_ADDRESS" env-default:"localhost:82"`
}

func MustLoad(configPath string) Config {
	var cfg Config
	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		log.Fatalf("cannot read config %q: %s", configPath, err)
	}
	return cfg
}
