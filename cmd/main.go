package main

import (
	"fmt"
	"log"

	"github.com/joaquinrovira/notes/internal/services/token"
)

func main() {
	TokenService, err := token.NewServiceFromEnv()
	if err != nil {
		log.Fatalf("Failed to init token service: %v", err)
	}

	token := token.NewTokenV1()
	token.Paths = []string{""}
	token.Index = "/auth/token"
	payload, err := TokenService.Encrypt(token)
	if err != nil {
		panic(err)
	}

	fmt.Printf("/auth/login?token=%s\n", payload)
}
