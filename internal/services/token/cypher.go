package token

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log"
	"os"
)

// Service handles encryption and decryption
type Service struct {
	key []byte
}

// NewService creates a token service with a 32-byte key
func NewService(hexKey string) (*Service, error) {
	if len(hexKey) != 32 {
		return nil, errors.New("encryption key must be exactly 32 bytes")
	}
	return &Service{key: []byte(hexKey)}, nil
}

func NewServiceFromEnv() (*Service, error) {
	secretKey := os.Getenv("ENCRYPTION_KEY")
	if secretKey == "" {
		log.Println("WARNING: Using default dev key. Do not use in production.")
		secretKey = "12345678901234567890123456789012"
	}
	return NewService(secretKey)
}

// Encrypt serializes, encrypts, and encodes the payload
func (s *Service) Encrypt(payload Token) (string, error) {
	// 1. Serialize to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	// 2. Create Cipher Block
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return "", err
	}

	// 3. Create GCM
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// 4. Create Nonce
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// 5. Encrypt (Seal)
	// format: nonce + ciphertext
	ciphertext := aesGCM.Seal(nonce, nonce, jsonData, nil)

	// 6. Base64 URL Encode
	return base64.URLEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decodes, decrypts, and deserializes the payload
func (s *Service) Decrypt(tokenString string) (Token, error) {
	// 1. Base64 Decode
	data, err := base64.URLEncoding.DecodeString(tokenString)
	if err != nil {
		return nil, err
	}

	// 2. Create Cipher Block
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return nil, err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := aesGCM.NonceSize()
	if len(data) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	// 3. Extract Nonce and Ciphertext
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]

	// 4. Decrypt (Open)
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	// 5. Unmarshal JSON
	return UnmarshalToken(plaintext)
}
