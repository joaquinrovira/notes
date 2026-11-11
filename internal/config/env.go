package config

import (
	"fmt"
	"os"
	"strings"
)

func Require(key string) (string, error) {
	key = strings.ToUpper(key)
	if v := os.Getenv(key); v == "" {
		return "", fmt.Errorf("missing enviroment variable: %s", key)
	} else {
		return v, nil
	}
}

func OrDefault(key, deflt string) string {
	k, err := Require(key)
	if err != nil {
		return deflt
	}
	return k
}
