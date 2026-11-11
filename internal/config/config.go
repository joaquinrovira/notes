package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

func New() (Config, error) {
	var zero Config
	dotenv, err := godotenv.Read()
	if err != nil {
		return zero, fmt.Errorf("reading environment from disk: %w", err)
	}

	return Config{
		dotenv: dotenv,
	}, nil
}

type Config struct {
	dotenv map[string]string
}

func (c Config) String(key string) (v string, ok bool) {
	if v, ok := os.LookupEnv(key); ok {
		return v, true
	}
	if v, ok := c.dotenv[key]; ok {
		return v, true
	}
	return "", false
}
