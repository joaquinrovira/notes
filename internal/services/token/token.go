package token

import (
	"encoding/json"
	"fmt"
)

type Token interface {
	payload()
}

func UnmarshalToken(data []byte) (Token, error) {
	var base tokenBase
	err := json.Unmarshal(data, &base)
	if err != nil {
		return nil, fmt.Errorf("invalid token base: %w", err)
	}

	switch base.Version {
	case tokenVersionV1:
		t := &TokenV1{}
		return t, json.Unmarshal(data, t)
	}

	return nil, fmt.Errorf("invalid token: unknown version \"%s\"", base.Version)
}

type tokenVersion string

const (
	tokenVersionAdmin tokenVersion = "0"
	tokenVersionV1    tokenVersion = "1"
)

var _ Token = (*tokenBase)(nil)

type tokenBase struct {
	Version tokenVersion `json:"v"`
}

func (*tokenBase) payload() {}
