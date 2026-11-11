package token

import (
	"encoding/json/v2"
	"fmt"
)

type Token interface {
	token()
}

var _ Token = (*tokenBase)(nil)

type tokenBase struct {
	Version tokenVersion `json:"v"`
}

func (*tokenBase) token() {}

type tokenVersion string

const (
	tokenVersionV1 tokenVersion = "1"
)

func UnmarshalToken(data []byte) (Token, error) {
	var base tokenBase
	err := json.Unmarshal(data, &base)
	if err != nil {
		return nil, fmt.Errorf("invalid token base: %w", err)
	}

	switch base.Version {
	case tokenVersionV1:
		t := NewTokenV1()
		return t, json.Unmarshal(data, t)
	}

	return nil, fmt.Errorf("invalid token: unknown version \"%s\"", base.Version)
}
