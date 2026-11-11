package token

import "time"

type TokenV1 struct {
	tokenBase
	Verified   bool       // Set to true after the token is verified during login
	NotBefore  *time.Time `json:"nbf"`    // Shows the [internal/countdown] timer
	NotAfter   *time.Time `json:"naf"`    // Logins will only work before this time
	Expiration *time.Time `json:"exp"`    // Token will expire after this time
	Home       string     `json:"home"`   // Default landing page
	Paths      []string   `json:"allows"` // Allowed path prefix
}

func NewTokenV1() *TokenV1 {
	return &TokenV1{
		Version: tokenVersionV1,
	}
}
