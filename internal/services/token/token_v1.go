package token

import "time"

type TokenV1 struct {
	tokenBase
	NotBefore  *time.Time `json:"nbf"`
	Expiration *time.Time `json:"exp"`
	Index      string     `json:"i"`     // Default landing page
	Paths      []string   `json:"paths"` // Allowed glob patterns
}

func NewTokenV1() *TokenV1 {
	return &TokenV1{
		tokenBase: tokenBase{Version: tokenVersionV1},
	}
}
