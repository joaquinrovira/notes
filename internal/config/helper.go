package config

import "fmt"

func Require(c Config, key string) (string, error) {
	v, ok := c.String(key)
	if !ok {
		return "", fmt.Errorf("missing configuration key: %s", key)
	}
	return v, nil
}

func OrDefault(cfg Config, key, deflt string) string {
	v, ok := cfg.String(key)
	if !ok {
		return deflt
	}
	return v
}

func OrZero(cfg Config, key string) string {
	return OrDefault(cfg, key, "")
}
