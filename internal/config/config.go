package config

import (
	"os"

	"github.com/joho/godotenv"
)

func New() (Config, error) {
	var c Config
	dotenv, err := godotenv.Read()
	if err == nil {
		c.dotenv = dotenv
	} else {
		c.dotenv = map[string]string{}
	}
	return c, nil
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
