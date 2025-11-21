package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
)

type token struct {
	id     string
	secret string
}

func randomString(byteNum int) string {
	id := make([]byte, byteNum)
	_, err := rand.Read(id)
	if err != nil {
		panic(err)
	}

	// Convert to hex string
	return hex.EncodeToString(id)
}

func newToken() token {
	return token{
		id:     randomString(4),
		secret: randomString(16),
	}
}

func (t token) ToTokenString() string {
	return fmt.Sprintf("%s.%s", t.id, t.secret)
}

func parseTokenString(tokenString string) (token, error) {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 2 {
		return token{}, fmt.Errorf("invalid token string")
	}
	return token{
		id:     parts[0],
		secret: parts[1],
	}, nil
}
