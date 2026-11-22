package auth

import (
	"crypto/sha256"
	"crypto/subtle"
	"fmt"

	"github.com/golang-jwt/jwt/v4"
)

type hashedToken struct {
	hash  [sha256.Size]byte
	value any
}

func AuthorizerStatic(tokens ...string) func(token string) (any, error) {
	values := make(map[string]any, len(tokens))
	for _, token := range tokens {
		if token != "" {
			values[token] = token
		}
	}

	return AuthorizerTokens(values)
}

func AuthorizerTokens(tokens map[string]any) func(token string) (any, error) {
	entries := make([]hashedToken, 0, len(tokens))
	for token, value := range tokens {
		if token == "" {
			continue
		}
		entries = append(entries, hashedToken{
			hash:  sha256.Sum256([]byte(token)),
			value: value,
		})
	}

	if len(entries) == 0 {
		return rejectAuthorizer
	}

	return func(token string) (any, error) {
		if token == "" {
			return nil, ErrorForbidden
		}

		actual := sha256.Sum256([]byte(token))
		var matched any
		found := false
		for _, entry := range entries {
			if subtle.ConstantTimeCompare(entry.hash[:], actual[:]) == 1 {
				matched = entry.value
				found = true
			}
		}

		if !found {
			return nil, ErrorForbidden
		}

		return matched, nil
	}
}

func AuthorizerJWT(signingKey []byte) func(token string) (any, error) {
	if len(signingKey) == 0 {
		return rejectAuthorizer
	}

	return AuthorizerJWTWithKeyFunc(func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Method.Alg())
		}

		return signingKey, nil
	})
}

func AuthorizerJWTWithKeyFunc(keyFunc jwt.Keyfunc) func(token string) (any, error) {
	if keyFunc == nil {
		return rejectAuthorizer
	}

	return func(tokenString string) (any, error) {
		token, err := jwt.Parse(tokenString, keyFunc)
		if err != nil {
			return nil, err
		}

		if !token.Valid {
			return nil, ErrorForbidden
		}

		return token, nil
	}
}
