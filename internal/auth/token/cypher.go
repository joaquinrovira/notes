package token

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
)

// Cypher handles encryption and decryption
type Cypher struct {
	cypher cipher.AEAD
}

// New creates a token service with a 32-byte key
func New(key string) (*Cypher, error) {
	if len(key) != 32 {
		return nil, errors.New("encryption key must be exactly 32 bytes")
	}

	// Create Cipher Block
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return nil, fmt.Errorf("creating AES cypher block: %w", err)
	}

	// Create GCM
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("creating GCM-AES cypher block: %w", err)
	}
	return &Cypher{cypher: aesGCM}, nil
}

// Encrypt serializes, encrypts, and encodes the payload
func (s *Cypher) Encrypt(payload Token) (string, error) {
	// Serialize to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	// Create Nonce
	nonce := make([]byte, s.cypher.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// Encrypt (Seal)
	// format: nonce + ciphertext
	ciphertext := s.cypher.Seal(nonce, nonce, jsonData, nil)

	// Base64 URL Encode
	return base64.URLEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decodes, decrypts, and deserializes the payload
func (s *Cypher) Decrypt(payload string) (Token, error) {
	// Base64 Decode
	data, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		return nil, err
	}

	nonceSize := s.cypher.NonceSize()
	if len(data) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	// Extract Nonce and Ciphertext
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]

	// Decrypt (Open)
	plaintext, err := s.cypher.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	// Unmarshal JSON
	return UnmarshalToken(plaintext)
}
