package database

import (
	"encoding/base64"
)

type rowScanner interface {
	Scan(...any) error
}

func base64Encode(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

func base64Decode(s string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(s)
}
